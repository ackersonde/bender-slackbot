package commands

// TestMessage is now commented
func checkCommand(slack*Api api, slack*Message slackMessage, string command) string {
  callingUserProfile, _ := api.GetUserInfo(slackMessage.Msg.User)
  
  return "whaddya say <@"+.Name+">? "+command+"?"
}