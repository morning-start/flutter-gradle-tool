package mirror

import "strings"

type Source struct {
	Name        string
	DisplayName string
	GradleURL   string
	MavenURL    string
}

var BuiltinSources = []Source{
	{
		Name:        "official",
		DisplayName: "Official (services.gradle.org)",
		GradleURL:   "https://services.gradle.org/distributions",
		MavenURL:    "",
	},
	{
		Name:        "tencent",
		DisplayName: "Tencent Cloud",
		GradleURL:   "https://mirrors.cloud.tencent.com/gradle",
		MavenURL:    "https://mirrors.cloud.tencent.com/nexus/repository/maven-public/",
	},
	{
		Name:        "aliyun",
		DisplayName: "Aliyun",
		GradleURL:   "https://mirrors.aliyun.com/maven/gradle",
		MavenURL:    "https://maven.aliyun.com/repository/public",
	},
	{
		Name:        "huaweicloud",
		DisplayName: "Huawei Cloud",
		GradleURL:   "https://mirrors.huaweicloud.com/gradle",
		MavenURL:    "https://mirrors.huaweicloud.com/repository/maven/",
	},
}

func FindByName(name string) *Source {
	normalized := normalizeName(name)
	if normalized == "" {
		return nil
	}

	for i := range BuiltinSources {
		if BuiltinSources[i].Name == normalized {
			return &BuiltinSources[i]
		}
	}
	return nil
}

func Names() []string {
	names := make([]string, 0, len(BuiltinSources))
	for i := range BuiltinSources {
		names = append(names, BuiltinSources[i].Name)
	}
	return names
}

func SourceFromDistributionURL(distributionURL string) *Source {
	if distributionURL == "" {
		return nil
	}

	normalizedURL := strings.ReplaceAll(distributionURL, `\`, "")
	for i := range BuiltinSources {
		if strings.Contains(normalizedURL, BuiltinSources[i].GradleURL) {
			return &BuiltinSources[i]
		}
	}
	return nil
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
