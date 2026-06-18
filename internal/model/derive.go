package model

// ComputeHealth applies the rollup rules (first match wins). PRs are
// informational and deliberately do not drive health in v1 (Dependabot PRs
// would make every repo yellow); CI, alerts, and undeployed changes do.
func ComputeHealth(r Repo) Health {
	if r.Archived {
		return HealthGray
	}
	// RED: something is broken or critically vulnerable.
	if r.CI == CIFailure || r.Dependabot.Critical > 0 || r.Dependabot.High > 0 {
		return HealthRed
	}
	// YELLOW: worth a look.
	switch {
	case r.CI == CIPending:
		return HealthYellow
	case !r.Untagged && r.Undeployed != 0: // -1 (>=1) or a positive exact count
		return HealthYellow
	case r.Dependabot.Moderate > 0 || r.Dependabot.Low > 0:
		return HealthYellow
	case r.CodeScanning > 0 || r.SecretScanning > 0:
		return HealthYellow
	}
	return HealthGreen
}

// UndeployedLabel renders the undeployed-changes cell.
func (r Repo) UndeployedLabel() string {
	switch {
	case r.Untagged:
		return "untagged"
	case r.Undeployed < 0:
		return ">=1"
	case r.Undeployed == 0:
		return "-"
	default:
		return itoa(r.Undeployed)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
