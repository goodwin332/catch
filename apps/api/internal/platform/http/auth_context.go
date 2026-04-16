package httpx

import "context"

type AuthenticatedUser struct {
	ID          string
	Email       string
	Username    string
	DisplayName string
	AvatarURL   string
	Rating      int
	Role        string
}

type authUserContextKey struct{}

func WithAuthenticatedUser(ctx context.Context, user AuthenticatedUser) context.Context {
	return context.WithValue(ctx, authUserContextKey{}, user)
}

func AuthenticatedUserFromContext(ctx context.Context) (AuthenticatedUser, bool) {
	user, ok := ctx.Value(authUserContextKey{}).(AuthenticatedUser)
	return user, ok
}
