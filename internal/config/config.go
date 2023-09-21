package config

type (
	Telegram struct {
		Token   string  `env:"TOKEN"`
		ChatIDs []int64 `env:"CHAT_IDS"`
	}

	Lea struct {
		// Basic data
		Citizenship        string `env:"CITIZENSHIP,required"`
		NumberOfApplicants int    `env:"NUMBER_OF_APPLICANTS,default=1"`
		LiveInBerlin       string `env:"LIVE_IN_BERLIN,default=yes"`

		// Appointment data
		MainReason  string `env:"MAIN_REASON,required"`
		Category    string `env:"CATEGORY,required"`
		Subcategory string `env:"SUBCATEGORY,required"`
	}

	AmtConfig struct {
		Telegram Telegram `env:",prefix=TELEGRAM_,required"`
		Lea      Lea      `env:",prefix=LEA_"`
	}
)
