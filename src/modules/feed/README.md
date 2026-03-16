# Feed Module

Responsibilities:
- Query organic posts (chronological + hotScore)
- Inject eligible survey posts (ads) unless user is ad-free
- Return cursor-based pages for `/feed/main`

Key tables:
- `Post`, `PostMetrics`, `SurveyCampaign`, `SurveyImpression`

