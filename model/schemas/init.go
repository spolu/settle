package schemas

import (
	"log"

	"github.com/spolu/peer-currencies/lib/errors"
	"github.com/spolu/peer-currencies/model"
)

func init() {
	go func() {
		err := model.CreateMintDBTables()
		if err != nil {
			log.Fatal(errors.Details(err))
		}
	}()
}
