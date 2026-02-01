# Rebalance v2 rollout plan

## Scope
- Update the rebalance menu to use group-level v2 signals and new rendering.
- Remove v1 rebalancing logic (service + CLI rendering) and any unused tests.
- Keep tests TDD: one test per checklist item, process in order.

## Checklist
- [x] CLI rebalance menu uses v2 group actions instead of v1 buy/sell recommendations
- [x] Rebalance CLI renderer shows group action table (BUY/NO_ACTION/SELL_CANDIDATE) with delta and manual-review flag
- [x] Rebalance menu handles missing price service and empty portfolio with v2 paths
- [x] Remove v1 recommendation methods from RebalanceService and update call sites
- [x] Remove v1 CLI rendering helpers and any unused constants/types
- [x] Update/replace tests to cover v2 menu output and remove v1 test coverage
- [x] Delete any dead imports/usages after v1 removal (lint-safe)
- [x] Verify full test suite passes after v2 switch

## Notes
- v2 source of truth: RebalanceService.get_group_actions_v2() + GroupRebalanceSignal.
- Manual review should be visible in the CLI output (e.g., Yes/No column).
- Deleting v1 means removing get_buy_recommendations/get_sell_recommendations and RebalanceRecommendation usage from the rebalance menu.
