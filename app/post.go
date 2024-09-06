package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/posthandler"
)

// NewPostHandler returns a no-op PostHandler
func NewPostHandler() (sdk.PostHandler, error) {
	return posthandler.NewPostHandler(
		posthandler.HandlerOptions{},
	)
}
