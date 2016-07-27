package trello

import (
	"fmt"
	"github.com/VojtechVitek/go-trello"
	"strings"
)

func (t TrelloIssueProvider) FetchData(done chan String) (result chan Issue, err chan error) {
  result := make(chan Issue)
	trello, err := trello.NewAuthClient(t.Configuration.ApiKey, &t.Configuration.Token)
	if err != nil {
		log.Fatal(err)
	}

	// @trello Boards
	board, err := trello.Board(t.BoardId)
	if err != nil {
		log.Fatal(err)
	}

	// @trello Board Lists
	lists, err := board.Lists()
	if err != nil {
		log.Fatal(err)
	}
	for _, list := range lists {
		if strings.Compare(list.Name, t.ListName) == 0 {
			// @trello Board List Cards
			cards, _ := list.Cards()
			for _, card := range cards {
				cardName := card.Name
				description := card.Desc
				issueInstance := Issue{cardName, description}
				result <- issueInstance
			}
			close(result)
		}
	}
}
