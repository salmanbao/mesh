# Event Contracts

Canonical event schemas used by mesh services.

## M01 Authentication Events
- `auth.2fa.required.json`
- `user.registered.json`
- `user.deleted.json`

## M02 Profile Events
- `user.profile_updated.json`

## Financial Rails Events
- `payout.paid.json`
- `payout.failed.json`
- `payout.processing.json`
- `reward.calculated.json`
- `reward.payout_eligible.json`

## Notification Dependencies (M03)
- `campaign.budget_updated.json`
- `campaign.created.json`
- `campaign.launched.json`
- `dispute.created.json`
- `submission.approved.json`
- `submission.rejected.json`
- `transaction.failed.json`

## Reward Engine Dependencies (M41)
- `submission.auto_approved.json`
- `submission.cancelled.json`
- `submission.verified.json`
- `submission.view_locked.json`
- `tracking.metrics.updated.json`
