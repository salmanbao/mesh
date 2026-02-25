# Documentation Workflow (Mesh)

## Step 1: Read Before Writing
- Read changed code in `mesh/services/<cluster>/<service>/internal/*`.
- Read related contracts and service README.
- Read canonical spec files for ownership and dependencies.

## Step 2: Capture Behavioral Facts
- List commands/queries exposed.
- List side effects: DB writes, event emits, remote calls.
- Identify state transitions and guard conditions.

## Step 3: Write/Update Docs
- Update service README first.
- Update `mesh/docs/` only when cross-service behavior changed.
- Add a decision rationale block for notable tradeoffs.

## Step 4: Consistency Check
- Ensure terms match code and specs.
- Ensure dependency lists match actual imports/calls.
- Ensure event names and ownership terms are canonical.