#!/usr/bin/env python3
"""
Interview Chatbot — Automated Eval Script
Runs a full session flow against the server and outputs a structured report.

Usage:
  ./scripts/eval.py --server http://localhost:9090 --domain domains/example_domain.yaml
  ./scripts/eval.py --server http://localhost:9090 --domain domains/example_domain.yaml --answers answers.txt
  ./scripts/eval.py --server http://localhost:9090 --domain domains/example_domain.yaml --interactive
"""

import argparse
import json
import sys
import time
import urllib.request
import urllib.error


def api_call(url, method="GET", body=None, timeout=120):
    data = json.dumps(body).encode() if body else None
    req = urllib.request.Request(url, data=data, method=method)
    req.add_header("Content-Type", "application/json")
    try:
        resp = urllib.request.urlopen(req, timeout=timeout)
        return json.loads(resp.read().decode())
    except urllib.error.HTTPError as e:
        body = e.read().decode()
        print(f"  HTTP {e.code}: {body}", file=sys.stderr)
        return None
    except Exception as e:
        print(f"  Error: {e}", file=sys.stderr)
        return None


def run_init(server, domain_config):
    url = f"{server}/sessions/init"
    print(f"\n▶ INIT: {domain_config}")
    t0 = time.time()
    data = api_call(url, method="POST", body={"domain_config_path": domain_config})
    elapsed = time.time() - t0
    if not data:
        print("  ✗ FAILED")
        return None
    print(f"  ✓ Session {data['session_id'][:8]}... ({elapsed:.1f}s)")
    print(f"  Pool: {len(data['pool'])} questions")
    for entry in data.get("generation_log", []):
        err = entry.get("error", "")
        if err:
            print(f"    ⚠ {entry['node_path']}/{entry['archetype']}: {err}")
        else:
            print(f"    ✓ {entry['node_path']}/{entry['archetype']}: {entry['count']} questions")
    return data


def run_turn(server, session_id, answer):
    url = f"{server}/sessions/{session_id}/turn"
    t0 = time.time()
    data = api_call(url, method="POST", body={"answer": answer})
    elapsed = time.time() - t0
    if not data:
        return None, elapsed
    return data, elapsed


def run_summary(server, session_id, narrative=True):
    url = f"{server}/sessions/{session_id}/summary"
    if narrative:
        url += "?include_narrative=true"
    t0 = time.time()
    data = api_call(url)
    elapsed = time.time() - t0
    if not data:
        return None, elapsed
    return data, elapsed


def print_question(num, question, total):
    status = ""
    if question:
        status = f" (asked)" if question.get("status") == "asked" else ""
    print(f"\n{'─' * 60}")
    print(f"  Q{num}/{total} [{question.get('archetype','?')}] {question.get('difficulty','?')}{status}")
    print(f"  {question.get('text', 'N/A')}")
    if question.get("ideal_answer_hint"):
        print(f"  Hint: {question['ideal_answer_hint']}")
    print(f"{'─' * 60}")


def print_eval(eval_data):
    if not eval_data:
        return
    print(f"\n  Score: {eval_data.get('score', '?')}/5  " +
          f"{'⚠ VAGUE' if eval_data.get('vague_flag') else '✓ Clear'}")
    if eval_data.get("concepts_covered"):
        print(f"  Covered: {', '.join(eval_data['concepts_covered'])}")
    if eval_data.get("missing"):
        print(f"  Missing: {', '.join(eval_data['missing'])}")
    print(f"  Reasoning: {eval_data.get('reasoning', 'N/A')}")


def auto_answer(question, round_num):
    archetype = question.get("archetype", "")
    text = question.get("text", "")
    if archetype == "reasoning":
        return "I would analyze the pattern and deduce the answer step by step."
    elif archetype == "follow_up":
        return "To elaborate further, I would consider additional factors such as the specific constraints and requirements of the system. The key is to balance trade-offs based on the use case."
    elif archetype == "case":
        return "I would design a distributed architecture using a message queue for ingestion, stream processing for real-time computation, and a scalable storage layer. For high availability, I'd use replication and auto-scaling groups behind a load balancer."
    else:
        return "This requires understanding the fundamental concepts. The best approach depends on the specific context and requirements. I would analyze the trade-offs and choose the most appropriate solution based on factors like performance, scalability, and maintainability."


def main():
    parser = argparse.ArgumentParser(description="Run automated interview eval")
    parser.add_argument("--server", default="http://localhost:9090", help="Server URL")
    parser.add_argument("--domain", default="domains/real_estate_readiness.yaml", help="Domain config path")
    parser.add_argument("--answers", help="File with answers (one per line)")
    parser.add_argument("--interactive", action="store_true", help="Prompt for each answer")
    parser.add_argument("--no-narrative", action="store_true", help="Skip narrative in summary")
    args = parser.parse_args()

    print("=" * 70)
    print("  INTERVIEW CHATBOT — Eval Script")
    print(f"  Server: {args.server}")
    print(f"  Config: {args.domain}")
    print("=" * 70)

    preloaded_answers = []
    if args.answers:
        with open(args.answers) as f:
            preloaded_answers = [line.strip() for line in f if line.strip()]

    init_data = run_init(args.server, args.domain)
    if not init_data:
        sys.exit(1)

    session_id = init_data["session_id"]
    pool = init_data["pool"]
    total_expected = len(pool)

    current_question = None
    for q in pool:
        if q["status"] == "asked":
            current_question = q
            break
    if not current_question:
        print("  ✗ No 'asked' question found in pool")
        sys.exit(1)

    round_num = 0
    results = []
    answer_idx = 0

    while current_question and round_num < 20:
        round_num += 1
        print_question(round_num, current_question, total_expected)

        if args.interactive:
            answer = input("  Your answer > ")
        elif preloaded_answers and answer_idx < len(preloaded_answers):
            answer = preloaded_answers[answer_idx]
            answer_idx += 1
            print(f"\n  Answer [{answer_idx}]: {answer[:80]}...")
        else:
            answer = auto_answer(current_question, round_num)
            print(f"\n  Answer (auto): {answer[:80]}...")

        turn_data, elapsed = run_turn(args.server, session_id, answer)
        if not turn_data:
            print("  ✗ Turn failed, aborting")
            break

        eval_result = turn_data.get("eval_result", {})
        print_eval(eval_result)

        results.append({
            "question": current_question,
            "answer": answer,
            "eval": eval_result,
            "elapsed": elapsed,
        })

        if turn_data.get("follow_up_fired"):
            print(f"  ➜ Follow-up fired (depth {turn_data.get('follow_up_depth', 0)})")

        current_question = turn_data.get("next_question")

        if turn_data.get("interview_done"):
            print(f"\n{'=' * 60}")
            print("  ✓ INTERVIEW COMPLETE")
            break

        if current_question:
            current_question["status"] = "asked"

    print(f"\n  Total rounds: {round_num}")
    print(f"  Elapsed per turn: {sum(r['elapsed'] for r in results):.1f}s")

    summary_data, s_elapsed = run_summary(args.server, session_id, not args.no_narrative)
    if summary_data:
        print(f"\n{'=' * 70}")
        print("  SUMMARY")
        print(f"{'=' * 70}")
        print(f"  Overall score: {summary_data.get('overall', 'N/A'):.2f}/5")
        for path, info in summary_data.get("leaf_scores", {}).items():
            print(f"    {path}: avg {info['average']} ({info['count']} answers)")

        if summary_data.get("narrative"):
            print(f"\n  Narrative Assessment:")
            for line in summary_data["narrative"].strip().split("\n"):
                print(f"    {line}")

        print(f"\n  Timing: summary={s_elapsed:.1f}s")

    print(f"\n{'=' * 70}")
    print("  DONE")
    print(f"{'=' * 70}")


if __name__ == "__main__":
    main()
