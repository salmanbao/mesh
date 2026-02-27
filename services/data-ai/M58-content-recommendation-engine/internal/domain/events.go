package domain

// Module-internal event names are defined in the M58 spec but are not present in the global canonical event registry.
// We still model them explicitly so outbox semantics and envelope invariants are enforced in-service.
