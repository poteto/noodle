# Events

Events are how the noodle loop tracks what happened and how external systems feed information into Noodle. The scheduler reads recent events alongside backlog state and decides how to react.

Noodle emits internal events automatically (stage completed, order failed, merge conflict). You can inject your own events from CI pipelines, deploy scripts, monitoring tools, or anything else.

## Emitting events

Use the CLI to send an event into the noodle loop:

```bash
noodle event emit <type> [--payload <json>]
```

The type can be any string. The payload is optional JSON with whatever context is relevant.

```bash
# CI pipeline failed
noodle event emit ci.failed --payload '{"build_url": "https://ci.example.com/123", "reason": "tests failed"}'

# Deploy completed
noodle event emit deploy.completed --payload '{"env": "prod", "version": "v1.2.3"}'

# Flaky test detected
noodle event emit test.flaky --payload '{"test": "test_login_timeout", "failures": 3}'
```

Events append to `.noodle/loop-events.ndjson`. The next time Noodle builds the mise, recent events appear in the `recent_events` array for the scheduler to read.

## Internal events

The noodle loop emits these automatically:

| Event type | When |
| --- | --- |
| `stage.completed` | A stage finished successfully |
| `stage.failed` | A stage failed |
| `order.completed` | All stages in an order finished |
| `order.failed` | An order failed terminally |
| `merge.conflict` | A worktree merge hit a conflict |
| `order.dropped` | An order was removed (task type no longer registered) |
| `order.requeued` | A failed order was reset for another attempt |
| `registry.rebuilt` | The skill registry changed (skills added or removed) |

## How the scheduler uses events

Events flow into the mise as context, not commands. The scheduler reads them and uses judgment:

- After `stage.failed`: does this need a debugging order, or should it retry with a different approach?
- After `order.completed`: is there follow-up work? Related items that were blocked?
- After `merge.conflict`: avoid re-scheduling immediately, it may need manual attention.
- After a custom `ci.failed`: maybe schedule an investigation order.

There is no mechanical trigger matching. The scheduler is an agent. It reads the events, reads your schedule skill instructions, and decides what to do. You can influence its behavior by writing guidance in your `schedule` skill about how to react to specific event types.

## Session events

Events can also target a specific agent session:

```bash
noodle event emit stage_message --session <session_id> --payload '{"message": "Implementation complete"}'
```

Session events write to `.noodle/sessions/<session_id>/events.ndjson` instead of the loop event log. These are used for agent-to-loop communication during a running stage.

## Retention

The loop event log keeps the last 200 records. The mise brief includes up to 50 events since the last scheduling run. Older events are trimmed automatically.
