package snakeCharm

import (
	"fmt"
        "strings"
        "errors"
        "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const StringType = "string"
const BoolType = "bool"
const IntType = "int"
const ParentType = "parent"

type ConfigEntry struct {
    Type string
    Name string
    Help string
    Required bool
    Default interface{}
    Children []*ConfigEntry
    prefix string
}

func (ce ConfigEntry ) IsString() bool {
    return ce.Type == StringType
}

func (ce ConfigEntry) IsParent() bool {
    return ce.Type == ParentType
}

func (ce ConfigEntry) IsBool() bool {
    return ce.Type == BoolType
}

func (ce ConfigEntry) IsInt() bool {
    return ce.Type == IntType
}

func (ce ConfigEntry) GetString() string {
    return ce.Default.(string)
}

func (ce ConfigEntry) GetInt() int {
    return ce.Default.(int)
}

func (ce ConfigEntry) GetBool() bool {
    return ce.Default.(bool)
}

func (ce ConfigEntry) FlagName() string {
    return strings.Replace(ce.GetName(), ".", "-", -1)
}

func (ce ConfigEntry) GetName() string {
    if ce.prefix == "" {
        return ce.Name
    } else {
        return ce.prefix + "." + ce.Name
    }
}

func (ce *ConfigEntry) SetPrefix( p string ) {
    ce.prefix = p
}

func addBool( cfg *viper.Viper, e ConfigEntry ) {
    pflag.Bool(e.FlagName(), false, e.Help)
    cfg.SetDefault( e.GetName(), e.GetBool() )
}

func addString( cfg *viper.Viper, e ConfigEntry ) {
    pflag.String(e.FlagName(), "", e.Help)
    cfg.SetDefault( e.GetName(), e.GetString() )
}

func addInt( cfg *viper.Viper, e ConfigEntry ) {
    pflag.Int(e.FlagName(), 0, e.Help)
    cfg.SetDefault( e.GetName(), e.GetInt() )
}

func addConfigChildren( cfg *viper.Viper, prefix string, config []*ConfigEntry ) *viper.Viper {
    for _, e := range( config ) {
        e.SetPrefix( prefix )
        if e.IsBool() {
            addBool( cfg, *e )
        } else if e.IsInt() {
            addInt( cfg, *e )
        } else if e.IsString() {
            addString( cfg, *e )
        } else if e.IsParent() {
            cfg = addConfigChildren( cfg, e.GetName(), e.Children )
        } else {
            panic(fmt.Sprintf("%s is of unknown type: %s", e.GetName(), e.Type))
        }

        cfg.BindPFlag(e.GetName(), pflag.Lookup(e.FlagName()))
    }

    return cfg
}

func verifyRequiredConfigs( cfg *viper.Viper, prefix string, config []*ConfigEntry ) []string {
    missingValues := []string{}
    for _, e := range( config ) {
        if e.IsParent() {
            missingValues = append( missingValues, verifyRequiredConfigs( cfg, e.GetName(), e.Children )... )
        } else if e.Required {
            if e.IsString() && cfg.GetString(e.GetName()) == "" {
                missingValues = append(missingValues, e.GetName())
            } else if e.IsInt() && cfg.GetInt(e.GetName()) == 0 {
                missingValues = append(missingValues, e.GetName())
            }
        }
    }

    return missingValues
}


func BuildConfig( cfg *viper.Viper, config []*ConfigEntry ) ( *viper.Viper, error ) {
    if cfg == nil {
        cfg = viper.New()
    }
    cfg.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
    cfg.AutomaticEnv()
    cfg.SetConfigName("config") // name of config file (without extension)
    cfg.AddConfigPath("/etc/hackbot2000")   // path to look for the config file in
    cfg.AddConfigPath(".")
    // Ignore errors because we can config via cmdline or env
    _ = cfg.ReadInConfig() // Find and read the config file

    cfg = addConfigChildren( cfg, "", config )

    pflag.Parse()

    missingValues := verifyRequiredConfigs( cfg, "", config )
    var err error
    if len( missingValues ) > 0 {
        err = errors.New(fmt.Sprintf("You must set these config itmes: %v", missingValues ))
    }

    return cfg, err
}

func main() {
    cfg, err := BuildConfig( nil, []*ConfigEntry{
        {
            Type: IntType,
            Name: "port",
            Help: "Port on which to listen for requests",
            Required: true,
            Default: 80,
        },
        {
            Type: ParentType,
            Name: "slack",
            Children: []*ConfigEntry{
                {
                    Type: StringType,
                    Name: "token",
                    Help: "Token to use when dealing with slack",
                    Required: true,
                    Default: "",
                },
            },
        },
        {
            Type: ParentType,
            Name: "db",
            Children: []*ConfigEntry{
                {
                    Type: IntType,
                    Name: "port",
                    Help: "Port on which to connect to the db",
                    Required: true,
                    Default: 3306,
                },
                {
                    Type: StringType,
                    Name: "host",
                    Help: "Hostname to connect to in the db",
                    Required: true,
                    Default: "localhost",
                },
                {
                    Type: StringType,
                    Name: "username",
                    Help: "Username to use when connecting to the DB",
                    Required: true,
                    Default: "",
                },
                {
                    Type: StringType,
                    Name: "password",
                    Help: "Password to use when connecting to the DB",
                    Required: true,
                    Default: "",
                },
                {
                    Type: StringType,
                    Name: "name",
                    Help: "Name of database to use",
                    Required: true,
                    Default: "",
                },
            },
        },
    })
    if err != nil {
        panic( err )
    }

    fmt.Printf("port: %d\n", cfg.GetInt("port"))
    fmt.Printf("db.port: %d\n", cfg.GetInt("db.port"))
    fmt.Printf("db.host: %s\n", cfg.GetString("db.host"))
    fmt.Printf("db.username: %s\n", cfg.GetString("db.username"))
    fmt.Printf("db.password: %s\n", cfg.GetString("db.password"))
    fmt.Printf("db.name: %s\n", cfg.GetString("db.name"))
    fmt.Printf("slack.token: %s\n", cfg.GetString("slack.token"))
}
