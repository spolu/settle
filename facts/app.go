package facts

import "goji.io"

// Build initializes the app and its web stack.
func Build() (*web.Mux, error) {
	mux := goji.NewMux()

	return mux, nil
}
