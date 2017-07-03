package trials

// Constant Trials Report API endpoints
const (
	TrialsBaseURL                  = "https://api.destinytrialsreport.com"
	TrialsCurrentMapEndpoint       = TrialsBaseURL + "/currentMap"
	TrialsCurrentWeekStatsEndpoint = TrialsBaseURL + "/maps/week/0"
	// Variable component is the membershipId for the player
	TrialsCurrentWeekEndpointFmt = TrialsBaseURL + "/currentWeek/%s"
	TrialsTopWeaponsEndpointFmt  = TrialsBaseURL + "/topWeapons/%s"
	//TrialsGetOpponentsEndpointFmt = TrialsBaseURL + "/getOpponents/%s"
	// Week Number
	TrialsWeaponPercentageEndpointFmt = TrialsBaseURL + "/leaderboard/percentage/%s"

	// How many weapons to return in the Alexa response describing usage stats
	TopWeaponUsageLimit = 3
)
