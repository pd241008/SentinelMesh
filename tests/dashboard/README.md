# Dashboard Tests

Next.js-based tests for the dashboard (Track 3).

## Status
- **Build passes** with 2 routes and 2 chart components
- Unit tests for data loaders and components **not yet written**
- Placeholder directory for future component, integration, and E2E tests

## Current implementation
| Route | Content |
|---|---|
| `/` | Sweep overview — recall chart (gossip/independent/centralized), bandwidth chart, raw results table |
| `/crosscheck` | ML crosscheck — overall metrics table + per-category breakdown |

## Planned test suites
| Suite | Focus |
|---|---|
| Unit | CSV parsing (`parseSweepCSV`), data transformation |
| Component | SweepChart, BandwidthChart rendering with mock data |
| E2E | Key user flows (navigation, data load states) |

## Verify build
```bash
cd dashboard
npm run build
```
