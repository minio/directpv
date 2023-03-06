// This file is part of MinIO Console Server
// Copyright (c) 2023 MinIO, Inc.
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

import { createSlice, PayloadAction } from "@reduxjs/toolkit";
import { ISessionResponse } from "./types";
import { RootState } from '../../app/store';


export interface ConsoleState {
  session: ISessionResponse;
}

const initialState: ConsoleState = {
  session: {
    status: "",
  },
};

export const consoleSlice = createSlice({
  name: "console",
  initialState,
  reducers: {
    saveSessionResponse: (state, action: PayloadAction<ISessionResponse>) => {
      state.session = action.payload;
    },
    resetSession: (state) => {
      state.session = initialState.session;
    },
  },
});

export const { saveSessionResponse, resetSession } = consoleSlice.actions;
export const session = (state: RootState) => state.console.session;

export default consoleSlice.reducer;
