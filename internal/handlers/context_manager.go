package handlers

import "context"

type ctxKey int

const (
	_ ctxKey = iota
	ctxKeyRequestUserData
	ctxKeyRequestFromProtected
)

type UserData struct {
	ID       int
	Username string
	Tzone    string
}

// withRequestUserData creates a new context that has UserData injected.
func withRequestUserData(
	ctx context.Context, requestData UserData,
) context.Context {

	return context.WithValue(ctx, ctxKeyRequestUserData, requestData)
}

// requestUserData tries to retrieve requestUserData from the given context.
// If it doesn't exist, an empty struct UserData is returned.
func requestUserData(ctx context.Context) UserData {
	if userData, ok := ctx.Value(ctxKeyRequestUserData).(UserData); ok {

		return userData
	}

	return UserData{}
}

// withRequestFromProtected creates a new context that has
// the fromProtected flag (bool) injected.
func withRequestFromProtected(
	ctx context.Context, requestFromProtected bool,
) context.Context {

	return context.WithValue(
		ctx, ctxKeyRequestFromProtected, requestFromProtected,
	)
}

// requestFromProtected tries to retrieve the fromProtected flag
// of the given context. If it does not exist,
// its default value is returned, that is, false.
func requestFromProtected(ctx context.Context) bool {
	if fromProtected, ok := ctx.Value(ctxKeyRequestFromProtected).(bool); ok {

		return fromProtected
	}

	return false
}
