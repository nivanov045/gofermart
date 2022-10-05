package log

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

func Init() {
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
}

func Panic(err error) {
	log.Error().Stack().Err(errors.Wrap(err, "from error")).Send()
	panic(err)
}

func Error(err error) {
	log.Error().Stack().Err(errors.Wrap(err, "from error")).Send()
}

func Warn(err error) {
	log.Warn().Stack().Err(errors.Wrap(err, "from error")).Send()
}

func Debug(msg string) {
	log.Debug().Stack().Msg(msg)
}
