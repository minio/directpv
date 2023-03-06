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

import { createSlice, PayloadAction } from '@reduxjs/toolkit';
import { RootState } from './app/store';
import { snackBarMessage } from "./types";
import { ErrorResponseHandler } from "./common/api/types";

const initSideBarOpen = localStorage.getItem("sidebarOpen")
  ? JSON.parse(localStorage.getItem("sidebarOpen")!)["open"]
  : true;

export interface SystemState {
  loggedIn: boolean;
  sidebarOpen: boolean;
  snackBar: snackBarMessage;
  modalSnackBar: snackBarMessage;
}

const initialState: SystemState = {
  loggedIn: false,
  sidebarOpen: initSideBarOpen,
  snackBar: {
    message: "",
    detailedErrorMsg: "",
    type: "message",
  },
  modalSnackBar: {
    message: "",
    detailedErrorMsg: "",
    type: "message",
  },
};

export const systemSlice = createSlice({
  name: "system",
  initialState,
  reducers: {
    userLogged: (state, action: PayloadAction<boolean>) => {
      state.loggedIn = action.payload;
    },
    menuOpen: (state, action: PayloadAction<boolean>) => {
      // persist preference to local storage
      localStorage.setItem(
        "sidebarOpen",
        JSON.stringify({ open: action.payload })
      );
      state.sidebarOpen = action.payload;
    },
    setSnackBarMessage: (state, action: PayloadAction<string>) => {
      state.snackBar = {
        message: action.payload,
        detailedErrorMsg: "",
        type: "message",
      };
    },
    setErrorSnackMessage: (
      state,
      action: PayloadAction<ErrorResponseHandler>
    ) => {
      state.snackBar = {
        message: action.payload.errorMessage,
        detailedErrorMsg: action.payload.detailedError,
        type: "error",
      };
    },
    setModalSnackMessage: (state, action: PayloadAction<string>) => {
      state.modalSnackBar = {
        message: action.payload,
        detailedErrorMsg: "",
        type: "message",
      };
    },
    setModalErrorSnackMessage: (
      state,
      action: PayloadAction<{ errorMessage: string; detailedError: string }>
    ) => {
      state.modalSnackBar = {
        message: action.payload.errorMessage,
        detailedErrorMsg: action.payload.detailedError,
        type: "error",
      };
    },
  },
});

// Reducer actions
export const {
  userLogged,
  menuOpen,
  setSnackBarMessage,
  setErrorSnackMessage,
  setModalErrorSnackMessage,
  setModalSnackMessage,
} = systemSlice.actions;

// Reducer selectors
export const loggedIn = (state: RootState) => state.system.loggedIn;

export default systemSlice.reducer;
