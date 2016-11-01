package schemas

import (
	"log"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/mint/model"
)

func init() {
	go func() {
		err := model.CreateMintDBTables()
		if err != nil {
			log.Fatal(errors.Details(err))
		}
	}()
}
