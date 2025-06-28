package services

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/safatanc/gsalt-core/internal/app/errors"
	"github.com/safatanc/gsalt-core/internal/app/models"
	"github.com/safatanc/gsalt-core/internal/infrastructures"
)

type ConnectService struct {
}

func NewConnectService() *ConnectService {
	return &ConnectService{}
}

func (s *ConnectService) GetCurrentUser(accessToken string) (*models.ConnectUser, error) {
	req, err := http.NewRequest("GET", infrastructures.Config.CONNECT_BASE_URL+"/users/me", nil)
	if err != nil {
		return nil, err
	}

	if accessToken == "" {
		return nil, errors.NewBadRequestError("Access token is required")
	}

	// Check if accessToken is Bearer token
	if strings.HasPrefix(accessToken, "Bearer ") {
		req.Header.Set("Authorization", accessToken)
	} else {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var webResponse models.WebResponse[models.ConnectUser]
	err = json.NewDecoder(resp.Body).Decode(&webResponse)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.NewAppError(resp.StatusCode, webResponse.Message)
	}

	return &webResponse.Data, nil
}

func (s *ConnectService) GetUser(connectId string) (*models.ConnectUser, error) {
	req, err := http.NewRequest("GET", infrastructures.Config.CONNECT_BASE_URL+"/users/"+connectId, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var webResponse models.WebResponse[models.ConnectUser]
	err = json.NewDecoder(resp.Body).Decode(&webResponse)
	if err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to decode response body")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.NewAppError(resp.StatusCode, webResponse.Message)
	}

	return &webResponse.Data, nil
}
