package main

type UserThresholds struct {
	Vul float64
	Mal float64
	Lic float64
	Eng float64
	Aut float64
}

type Package struct {
	Name               string  `json:"name"`
	Version            string  `json:"version"`
	Status             string  `json:"status"`
	LastUpdated        int64   `json:"last_updated"`
	License            string  `json:"license"`
	PackageScore       float64 `json:"package_score"`
	NumDependencies    int     `json:"num_dependencies"`
	NumVulnerabilities int     `json:"num_vulnerabilities"`
	Type               string  `json:"type"`
	RiskVectors        struct {
		Engineering   float64 `json:"engineering"`
		Vulnerability float64 `json:"vulnerability"`
		Author        float64 `json:"author"`
		MaliciousCode float64 `json:"malicious_code"`
		License       float64 `json:"license"`
	} `json:"riskVectors"`
	Dependencies interface{}  `json:"dependencies"`
	Vulnerabilities []struct {
		Cve         []string `json:"cve"`
		Severity    float64  `json:"severity"`
		RiskLevel   string   `json:"risk_level"`
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Remediation string   `json:"remediation"`
	} `json:"vulnerabilities"`
	Issues       []struct {
		Title	string `json:"title"`
		Description string `json:"description"`
		RiskLevel string `json:"risk_level"`
		RiskDomain string `json:"risk_domain"`
	}`json:"issues"`
}

type PhylumJson struct {
	JobID         string  `json:"job_id"`
	Ecosystem     string  `json:"ecosystem"`
	UserID        string  `json:"user_id"`
	UserEmail     string  `json:"user_email"`
	CreatedAt     int64   `json:"created_at"`
	Status        string  `json:"status"`
	Score         float64 `json:"score"`
	Pass          bool    `json:"pass"`
	Msg           string  `json:"msg"`
	Action        string  `json:"action"`
	NumIncomplete int     `json:"num_incomplete"`
	LastUpdated   int64   `json:"last_updated"`
	Project       string  `json:"project"`
	ProjectName   string  `json:"project_name"`
	Label         string  `json:"label"`
	Thresholds    struct {
		Author        float64 `json:"author"`
		Engineering   float64 `json:"engineering"`
		License       float64 `json:"license"`
		Malicious     float64 `json:"malicious"`
		Total         float64 `json:"total"`
		Vulnerability float64 `json:"vulnerability"`
	} `json:"thresholds"`
	Packages []Package `json:"packages"`
}

type pkgVerTuple struct {
	name string
	version string
}

