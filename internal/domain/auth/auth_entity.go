package auth

type TokenPair struct {
	AccessToken  string `json:"access_token"`  //Живет меньше
	RefreshToken string `json:"refresh_token"` //Живет дольше
}
