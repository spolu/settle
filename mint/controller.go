package mint

import (
	"net/http"

	"golang.org/x/net/context"
)

type controller struct{}

func (c *controller) CreateAsset(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
}
