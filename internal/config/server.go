package config

// ServerConfig maps directly to Arma Reforger's config.json format.
// Reference: https://community.bistudio.com/wiki/Arma_Reforger:Server_Config
type ServerConfig struct {
	BindAddress   string    `json:"bindAddress"`
	BindPort      int       `json:"bindPort"`
	PublicAddress string    `json:"publicAddress"`
	PublicPort    int       `json:"publicPort"`
	A2S           A2S       `json:"a2s"`
	RCON          RCON      `json:"rcon"`
	Game          Game      `json:"game"`
	Operating     Operating `json:"operating"`
}

type A2S struct {
	Address string `json:"address"`
	Port    int    `json:"port"`
}

type RCON struct {
	Address  string `json:"address,omitempty"`
	Port     int    `json:"port,omitempty"`
	Password string `json:"password,omitempty"`
}

type Game struct {
	Name               string         `json:"name"`
	Password           string         `json:"password"`
	PasswordAdmin      string         `json:"passwordAdmin"`
	Admins             []string       `json:"admins"`
	ScenarioID         string         `json:"scenarioId"`
	MaxPlayers         int            `json:"maxPlayers"`
	Visible            bool           `json:"visible"`
	CrossPlatform      bool           `json:"crossPlatform"`
	SupportedPlatforms []string       `json:"supportedPlatforms"`
	GameProperties     GameProperties `json:"gameProperties"`
	Mods               []Mod          `json:"mods"`
}

type GameProperties struct {
	ServerMaxViewDistance    int                    `json:"serverMaxViewDistance"`
	ServerMinGrassDistance   int                    `json:"serverMinGrassDistance"`
	NetworkViewDistance      int                    `json:"networkViewDistance"`
	DisableThirdPerson       bool                   `json:"disableThirdPerson"`
	FastValidation           bool                   `json:"fastValidation"`
	BattlEye                 bool                   `json:"battlEye"`
	VONDisableUI             bool                   `json:"VONDisableUI"`
	VONDisableDirectSpeechUI bool                   `json:"VONDisableDirectSpeechUI"`
	MissionHeader            map[string]interface{} `json:"missionHeader"`
}

type Mod struct {
	ModID   string `json:"modId"`
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type Operating struct {
	LobbyPlayerSynchronise bool      `json:"lobbyPlayerSynchronise"`
	JoinQueue              JoinQueue `json:"joinQueue"`
}

type JoinQueue struct {
	MaxSize int `json:"maxSize"`
}

// EveronGameMasterScenarioID is the scenario ID for the built-in Everon Game Master mission
const EveronGameMasterScenarioID = "{ECC61978EDCC2B5A}Missions/23_Campaign.conf"

// DefaultServerConfig returns a new ServerConfig populated with sensible defaults
func DefaultServerConfig(name, bindAddress, publicAddress string, gamePort, queryPort, maxPlayers int, adminPassword, gamePassword string) *ServerConfig {
	return &ServerConfig{
		BindAddress:   bindAddress,
		BindPort:      gamePort,
		PublicAddress: publicAddress,
		PublicPort:    gamePort,
		A2S: A2S{
			Address: "0.0.0.0",
			Port:    queryPort,
		},
		RCON: RCON{},
		Game: Game{
			Name:               name,
			Password:           gamePassword,
			PasswordAdmin:      adminPassword,
			Admins:             []string{},
			ScenarioID:         EveronGameMasterScenarioID,
			MaxPlayers:         maxPlayers,
			Visible:            true,
			CrossPlatform:      false,
			SupportedPlatforms: []string{"PLATFORM_PC"},
			GameProperties: GameProperties{
				ServerMaxViewDistance:    1600,
				ServerMinGrassDistance:   50,
				NetworkViewDistance:      500,
				DisableThirdPerson:       false,
				FastValidation:           true,
				BattlEye:                 true,
				VONDisableUI:             false,
				VONDisableDirectSpeechUI: false,
				MissionHeader:            map[string]interface{}{},
			},
			Mods: []Mod{},
		},
		Operating: Operating{
			LobbyPlayerSynchronise: true,
			JoinQueue: JoinQueue{
				MaxSize: 0,
			},
		},
	}
}
