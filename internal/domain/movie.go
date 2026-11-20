package domain


type Movie struct {
	ID        int64  `json:"id"`
	Year      int    `json:"year"`
	Title     string `json:"title"`
	Studios   string `json:"studios"`
	Producers string `json:"producers"`
	Winner    bool   `json:"winner"`
}


type Filter struct {
	Year   *int
	Winner *bool
}


type Interval struct {
	Producer     string `json:"producer"`
	Interval     int    `json:"interval"`
	PreviousWin  int    `json:"previousWin"`
	FollowingWin int    `json:"followingWin"`
}


type IntervalsResponse struct {
	Min []Interval `json:"min"`
	Max []Interval `json:"max"`
}
