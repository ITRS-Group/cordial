package instance

import (
	"fmt"
	"os"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// JavaGatewayCommand builds the JVM command and environment shared by the
// Java-based gateway components.
func JavaGatewayCommand(i geneos.Instance, skipFileCheck bool, defaultMainClass string) (args, env []string, home string, err error) {
	if i == nil {
		return nil, nil, "", os.ErrInvalid
	}

	cf := i.Config()
	home = i.Home()
	base := BaseVersion(i)
	jarPath := i.Host().Join(base, config.Get[string](cf, "jar"))
	pluginsDir := i.Host().Join(base, "plugins")
	classpath := jarPath + ":" + i.Host().Join(pluginsDir, "*")
	setup := PathTo(i, "setup")
	mainClass := config.Get[string](cf, "main-class",
		config.PromoteFrom("mainclass"), config.DefaultValue(defaultMainClass))

	args = []string{
		"--enable-native-access=ALL-UNNAMED",
		"-Djava.net.preferIPv4Stack=true",
	}
	logback := config.Get[string](cf, "logback")
	if logback != "" {
		args = append(args, "-Dlogback.configurationFile="+logback)
	}
	args = append(args,
		"-Xms"+strings.TrimPrefix(config.Get[string](cf, "xms", config.DefaultValue("512m")), "-Xms"),
		"-Xmx"+strings.TrimPrefix(config.Get[string](cf, "xmx", config.DefaultValue("512m")), "-Xmx"),
		"-XX:+UseG1GC",
		"-Dapp.home="+home,
	)
	args = append(args, strings.Fields(config.Get[string](cf, "java-options"))...)
	args = append(args, "-cp", classpath, mainClass, setup)
	env = []string{"JAVA_HOME=" + i.Host().Join(base, "jdk")}

	checks := []string{i.Host().Dir(LogFilePath(i)), jarPath, pluginsDir, setup}
	if logback != "" {
		checks = append(checks, logback)
	}
	if !skipFileCheck {
		if missing := CheckPaths(i, checks...); len(missing) > 0 {
			err = fmt.Errorf("%w: %v", os.ErrNotExist, missing)
		}
	}
	return
}
