package application

const (
	// eventTypeUserRegistered is emitted when a user account is created.
	eventTypeUserRegistered = "user.registered"
	// eventTypeUserDeleted is emitted when a user requests account deletion.
	eventTypeUserDeleted = "user.deleted"
	// eventTypeAuth2FARequired is emitted when primary auth requires a second factor.
	eventTypeAuth2FARequired = "auth.2fa.required"
)
