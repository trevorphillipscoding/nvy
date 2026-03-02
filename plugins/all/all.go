// Package all imports every built-in plugin so their init() functions run
// and they register themselves with the plugin registry.
//
// To add a new language runtime:
//  1. Create plugins/<lang>/<lang>.go implementing plugins.Plugin
//  2. Call plugins.Register(New()) in that package's init()
//  3. Add a blank import here
package all

import (
	_ "github.com/trevorphillipscoding/nvy/plugins/golang"
	_ "github.com/trevorphillipscoding/nvy/plugins/node"
	_ "github.com/trevorphillipscoding/nvy/plugins/python"
)
