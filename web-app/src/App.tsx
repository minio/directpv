import React from 'react';
import './App.css';
import {GlobalStyles, ThemeHandler} from "mds";
import {useSelector} from "react-redux";
import {RootState} from "./app/store";

const App = () => {

    const darkMode = useSelector((state: RootState) => state.theme.darkMode)

    return (
        <ThemeHandler darkMode={darkMode}>
            <GlobalStyles/>

        </ThemeHandler>
    );
}

export default App;
