import { createAsyncThunk, createSlice, PayloadAction } from '@reduxjs/toolkit';

export interface AuthState {
  status: 'idle' | 'loading' | 'failed';
}

const initialState: AuthState = {
  status: 'idle'
};

export const API_URL =
  window.location.port === '3000'
    ? "//" + window.location.hostname + ":8080"
    : "//" + window.location.host + "";

function loginRequest(password: string) {
  let formData: FormData = new FormData();
  formData.append('username', 'admin')
  formData.append('password', password)
  const init: RequestInit = {
    credentials: 'include',
    method: 'POST',
    mode: 'cors',
    body: formData,
  }
  const result = fetch(`${API_URL}/api/login`,init)
    .then(response => {
      return response.json()
    })
  return result
}

function logoutRequest() {
  const init: RequestInit = {
    credentials: 'include',
    mode: 'cors'
  }
  const result = fetch(`${API_URL}/api/logout`,init)
    .then(response => {
      return response.json()
    })
  return result
}

// The function below is called a thunk and allows us to perform async logic. It
// can be dispatched like a regular action: `dispatch(incrementAsync(10))`. This
// will call the thunk with the `dispatch` function as the first argument. Async
// code can then be executed and other actions can be dispatched. Thunks are
// typically used to make async requests.
export const loginAsync = createAsyncThunk(
  'auth/loginRequest',
  async (data: {password: string}) => {
    const response = await loginRequest(data.password);
    return response
  }
);
export const logoutAsync = createAsyncThunk(
  'auth/logoutRequest',
  async () => {
    const response = await logoutRequest();
    return response
  }
);


export const authSlice = createSlice({
  name: 'auth',
  initialState,
  // The `reducers` field lets us define reducers and generate associated actions
  reducers: {},
  // The `extraReducers` field lets the slice handle actions defined elsewhere,
  // including actions generated by createAsyncThunk or in other slices.
  extraReducers: (builder) => {
    builder
      .addCase(loginAsync.pending, (state) => {
        state.status = 'loading';
      })
      .addCase(loginAsync.fulfilled, (state, action) => {
        state.status = 'idle';
      });
  },
});

export default authSlice.reducer;
