package facts

import "goji.io"

// Build initializes the app and its web stack.
func Build() (*goji.Mux, error) {
	mux := goji.NewMux()

	return mux, nil
}
