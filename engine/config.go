package engine

import (
	"encoding/json"
	"log"
	"os"
)

var DefaultConfig = Config{
	SessionTokens: []string{
		"-tK98TNwSypCiQuoyN2lpnavQgdE_59XcRoCrSZcSo3g7QmzrYsuk_aZUv0J_95_",
	},
	Port:      8080,
	AutoStart: false,
	DefaultTriggers: DefaultTriggers{
		Claim:    "if you'd like to claim it use the",
		Unclaim:  "unclaimed this ticket",
		Reopened: "and re-opened it to you",
		Raffle:   "has been claimed",
	},
	Servers: map[string]ServerConfig{
		"01JZ61Q8WN45VQ0ZMCM59T10ZX": {
			Name:                "eo",
			UseTicketNumber:     false,
			CategoryNamePattern: "ticket",
			Messages:            []string{".j4"},
			UnclaimReply:        "",
			RaffleReply:         ".j4",
			AggressiveMode:      false,
		},
		"01KD6B9A08JM600VSDG4SHVFRV": {
			Name:                "Treatz",
			UseTicketNumber:     false,
			CategoryNamePattern: "ticket",
			Messages:            []string{"/claim"},
			UnclaimReply:        "",
			RaffleReply:         ".j4",
			AggressiveMode:      false,
		},
		"01JDKH82R0RHG2VF9YDWKEFHC5": {
			Name:                "CE",
			UseTicketNumber:     false,
			CategoryNamePattern: "ticket",
			Messages:            []string{"/claim"},
			UnclaimReply:        "",
			RaffleReply:         "",
			AggressiveMode:      false,
		},
		"01JDKAFHS1W2BTPSS9YDB6WNEP": {
			Name:                "goons",
			UseTicketNumber:     false,
			CategoryNamePattern: "ticket-14",
			Messages:            []string{".j4"},
			UnclaimReply:        "/claim",
			RaffleReply:         "",
			AggressiveMode:      false,
		},
		"01JDKH7HVTBZ2SDTYMTESVDEZA": {
			Name:                "GE",
			UseTicketNumber:     false,
			CategoryNamePattern: "ticket",
			Messages:            []string{"/claim", ".join"},
			UnclaimReply:        "",
			RaffleReply:         "",
			AggressiveMode:      false,
		},
		"01JDKJ2C7GRNTP9KCQZJWWQ6S0": {
			Name:                "foodie",
			UseTicketNumber:     false,
			CategoryNamePattern: "ticket",
			Messages:            []string{"/claim"},
			UnclaimReply:        "",
			RaffleReply:         ".j4",
			AggressiveMode:      false,
		},
		"01JDKZ9Y7AEPQQDA7BVQA10DZ7": {
			Name:                "border",
			UseTicketNumber:     false,
			CategoryNamePattern: "ticket",
			Messages:            []string{"/claim", ".join"},
			UnclaimReply:        "",
			RaffleReply:         "",
			AggressiveMode:      false,
		},
		"01K7A7CNSMC5XPTJ7J36H9XKGR": {
			Name:                "foodocity",
			UseTicketNumber:     true,
			CategoryNamePattern: "ticket",
			Messages:            []string{},
			UnclaimReply:        "",
			RaffleReply:         "",
			AggressiveMode:      false,
		},
		"01K7A7TBZ4SJKNXX47H9MHF6V7": {
			Name:                "tgc",
			UseTicketNumber:     true,
			CategoryNamePattern: "doordash",
			Messages:            []string{},
			UnclaimReply:        "",
			RaffleReply:         "",
			AggressiveMode:      false,
		},
		"01JDPY161J6H6B1KBV74QWKCDM": {
			Name:                "blackjack",
			UseTicketNumber:     false,
			CategoryNamePattern: "ticket",
			Messages:            []string{".j4"},
			UnclaimReply:        "",
			RaffleReply:         "",
			AggressiveMode:      false,
		},
	},
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig
			for k, v := range cfg.Servers {
				srv := v
				FillTriggers(&srv, cfg.DefaultTriggers)
				cfg.Servers[k] = srv
			}
			if saveErr := SaveConfig(path, &cfg); saveErr != nil {
				log.Printf("Failed to save default config: %v", saveErr)
			}
			return &cfg, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.DefaultTriggers.Claim == "" {
		cfg.DefaultTriggers = DefaultConfig.DefaultTriggers
	}
	for k, v := range cfg.Servers {
		srv := v
		FillTriggers(&srv, cfg.DefaultTriggers)
		cfg.Servers[k] = srv
	}
	if cfg.Port == 0 {
		cfg.Port = 8080
	}

	return &cfg, nil
}

func SaveConfig(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func LoadClaimed(path string) (map[string]bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]bool), nil
		}
		return nil, err
	}
	var ids []string
	if err := json.Unmarshal(data, &ids); err != nil {
		return nil, err
	}
	m := make(map[string]bool, len(ids))
	for _, id := range ids {
		m[id] = true
	}
	return m, nil
}

func SaveClaimed(path string, m map[string]bool) error {
	ids := make([]string, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	data, err := json.MarshalIndent(ids, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
