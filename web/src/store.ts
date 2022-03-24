import { configureStore } from '@reduxjs/toolkit'
// import { createStore } from 'redux';

// type storeType = { navHidden: boolean }

// const initialState: storeType = {
//   navHidden: false
// }

// const showNavReducer = (state: storeType = initialState, action: any) => {
//   if (action.type === 'toggleNav') {
//     return { navHidden: !state.navHidden }
//   }

//   return state;
// };

// const store = createStore(showNavReducer);

// export default store;


const store = configureStore({
  reducer: {
    "tableSlice": slice
  },
})

// Infer the `RootState` and `AppDispatch` types from the store itself
export type RootState = ReturnType<typeof store.getState>
// Inferred type: {posts: PostsState, comments: CommentsState, users: UsersState}
export type AppDispatch = typeof store.dispatch

export default store
