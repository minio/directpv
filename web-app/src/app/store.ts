import {Action, configureStore, ThunkAction} from '@reduxjs/toolkit';
import systemSlice from '../systemSlice';
import themeSlice from "../themeSlice";
import consoleSlice from "../screens/console/consoleSlice";

export const store = configureStore({
    reducer: {
        theme: themeSlice,
        system: systemSlice,
        console: consoleSlice,
    },
});

export type AppDispatch = typeof store.dispatch;
export type RootState = ReturnType<typeof store.getState>;
export type AppThunk<ReturnType = void> = ThunkAction<
    ReturnType,
    RootState,
    unknown,
    Action<string>
>;
