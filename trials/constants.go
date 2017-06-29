package trials

// Constant Trials Report API endpoints
const (
	TrialsBaseURL            = "https://api.destinytrialsreport.com"
	TrialsCurrentMapEndpoint = TrialsBaseURL + "/currentMap"
	// Variable component is the membershipId for the player
	TrialsCurrentWeekEndpointFmt  = TrialsBaseURL + "/currentWeek/%s"
	TrialsTopWeaponsEndpointFmt   = TrialsBaseURL + "/topWeapons/%s"
	TrialsGetOpponentsEndpointFmt = TrialsBaseURL + "/getOpponents/%s"
	// Week Number
	TrialsWeaponPercentageEndpointFmt = TrialsBaseURL + "/leaderboard/percentage/%d"
)
