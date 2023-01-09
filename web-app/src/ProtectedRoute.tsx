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

import React, { useEffect, useState } from "react";
import { Navigate, useLocation } from "react-router-dom";
import api from "./common/api";
import { ISessionResponse } from "./screens/console/types";
import { userLogged, } from "./systemSlice";
import { saveSessionResponse } from "./screens/console/consoleSlice";
import { useAppDispatch, useAppSelector } from "./app/hooks";

interface ProtectedRouteProps {
  Component: any;
}

const ProtectedRoute = ({ Component }: ProtectedRouteProps) => {
  const dispatch = useAppDispatch();


  const [sessionLoading, setSessionLoading] = useState<boolean>(true);
  const userLoggedIn = useAppSelector((state) => state.system.loggedIn);

  const { pathname = "" } = useLocation();

  const StorePathAndRedirect = () => {
    localStorage.setItem("redirect-path", pathname);
    return <Navigate to={{ pathname: `login` }} />;
  };

  useEffect(() => {
    api
      .invoke("GET", `/api/v1/session`)
      .then((res: ISessionResponse) => {
        dispatch(saveSessionResponse(res));
        dispatch(userLogged(true));
        setSessionLoading(false);

      })
      .catch(() => setSessionLoading(false));
  }, [dispatch]);

  if (sessionLoading) {
    console.log("implement loading component");
    // return <LoadingComponent />;
  }
  return userLoggedIn ? <Component /> : <StorePathAndRedirect />;
};

export default ProtectedRoute;
