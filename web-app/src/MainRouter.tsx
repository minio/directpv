import React from 'react';
import {createBrowserRouter, RouterProvider} from "react-router-dom";
import Login from "./screens/login/Login";
import App from "./App";
import ProtectedRoute from './ProtectedRoute';

const router = createBrowserRouter([
    {
        path: "/login",
        element: <Login/>,
    },
    {
        path: "/*",
        element: <ProtectedRoute Component={App} />,
    },
]);

const MainRouter = () => {
    return <RouterProvider router={router}/>
}

export default MainRouter;