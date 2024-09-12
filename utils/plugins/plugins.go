package plugins

//ai gen template for plugins.go
import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
	lua "github.com/yuin/gopher-lua"
)

type Plugin struct {
	Name string
	Path string
}

func LoadPlugins(e *echo.Echo) {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting current working directory: %v", err)
	}

	pluginDir := filepath.Join(dir, "plugins")
	files, err := os.ReadDir(pluginDir)
	if err != nil {
		log.Fatalf("Error reading plugin directory: %v", err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".lua") {
			luaFile := filepath.Join(pluginDir, file.Name())
			if err := loadAndRegisterPlugin(e, luaFile); err != nil {
				log.Printf("Failed to load and register plugin %s: %v", file.Name(), err)
			}
		}
	}
}

func loadAndRegisterPlugin(e *echo.Echo, luaFile string) error {
	L := lua.NewState()
	defer L.Close()

	if err := L.DoFile(luaFile); err != nil {
		return fmt.Errorf("error executing Lua file %s: %w", luaFile, err)
	}

	pluginName := strings.TrimSuffix(filepath.Base(luaFile), ".lua")
	pl := &Plugin{Name: pluginName, Path: luaFile}

	if err := pl.Register(e, L); err != nil {
		return fmt.Errorf("error registering plugin %s: %w", pl.Name, err)
	}

	fmt.Printf("Loaded and registered plugin %s\n", pl.Name)
	return nil
}

func (p *Plugin) Register(e *echo.Echo, L *lua.LState) error {
	fmt.Printf("Registering plugin %s\n", p.Name)

	register := L.GetGlobal("Register")
	if register == lua.LNil {
		return fmt.Errorf("register function not found in plugin %s", p.Name)
	}

	if err := L.CallByParam(lua.P{
		Fn:      register,
		NRet:    0,
		Protect: true,
	}); err != nil {
		return fmt.Errorf("error calling Register function in plugin %s: %w", p.Name, err)
	}

	return nil
}
