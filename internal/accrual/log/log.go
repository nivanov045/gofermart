package log

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

func Init() {
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
}

func Panic(err error) {
	log.Error().Stack().Err(err).Send()
	panic(err)
}

func Error(err error) {
	log.Error().Stack().Err(err).Send()
}

func Warn(err error) {
	log.Warn().Stack().Err(err).Send()
}

func Debug(msg string) {
	log.Debug().Stack().Msg(msg)
}
