package contracts

type SuccessResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

type ErrorPayload struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

type ErrorResponse struct {
	Status string       `json:"status"`
	Error  ErrorPayload `json:"error"`
}

type CreateTeamRequest struct {
	ScopeType string `json:"scope_type"`
	ScopeID   string `json:"scope_id"`
}

type CreateTeamResponse struct {
	TeamID  string `json:"team_id"`
	OwnerID string `json:"owner_id"`
	Status  string `json:"status"`
}

type CreateInviteRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type CreateInviteResponse struct {
	InviteID  string `json:"invite_id"`
	Status    string `json:"status"`
	ExpiresAt string `json:"expires_at"`
}

type AcceptInviteResponse struct {
	TeamID     string `json:"team_id"`
	MemberRole string `json:"member_role"`
	Status     string `json:"status"`
}

type TeamMemberDTO struct {
	UserID   string `json:"user_id"`
	Role     string `json:"role"`
	JoinedAt string `json:"joined_at"`
}

type InviteDTO struct {
	InviteID  string `json:"invite_id"`
	Status    string `json:"status"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	ExpiresAt string `json:"expires_at"`
}

type TeamDetailsResponse struct {
	TeamID  string          `json:"team_id"`
	Members []TeamMemberDTO `json:"members"`
	Invites []InviteDTO     `json:"invites"`
}

type MembershipResponse struct {
	Allowed bool   `json:"allowed"`
	Role    string `json:"role,omitempty"`
}
