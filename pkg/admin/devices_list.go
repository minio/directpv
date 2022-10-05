// This file is part of MinIO DirectPV
// Copyright (c) 2021, 2022 MinIO, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ListDevices - lists the available and unavailable devices present in the nodes. Also honours the selectors for filtering
func (adm *Client) ListDevices(ctx context.Context, req GetDevicesRequest) (result GetDevicesResponse, err error) {
	reqBodyInBytes, err := json.Marshal(req)
	if err != nil {
		return result, fmt.Errorf("errror while marshalling the request: %v", err)
	}
	resp, err := adm.executeMethod(ctx, "", RequestData{
		RelPath: devicesListAPIPath,
		Content: reqBodyInBytes,
	})
	defer closeResponse(resp)
	if err != nil {
		return result, err
	}
	if resp.StatusCode != http.StatusOK {
		return result, httpRespToErrorResponse(resp)
	}
	responseInBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, fmt.Errorf("failed to read response: %v", err)
	}
	if err := json.Unmarshal(responseInBytes, &result); err != nil {
		return result, fmt.Errorf("failed to parse the response: %v", err)
	}
	return
}
