package command

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeletePersonalChanelWelcomeValidate(t *testing.T) {
	setup := func() *DeletePersonalChanelWelcome {
		return &DeletePersonalChanelWelcome{}
	}

	t.Run("happy path", func(t *testing.T) {
		c := setup()
		err := c.Validate(make([]string, 0))

		assert.NoError(t, err)
	})

	t.Run("with extra parameters", func(t *testing.T) {
		c := setup()
		err := c.Validate([]string{"a"})

		assert.Error(t, err)
	})
}
