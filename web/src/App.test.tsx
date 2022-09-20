import { render, screen } from "@testing-library/react";
import App from "./App";
import { Provider } from "react-redux";
import { store } from "./store/store";

test("renders login page", () => {
  render(
    <Provider store={store}>
      <App />
    </Provider>
  );
  const elemane = screen.getByText(/login/i);
  expect(elemane).toBeInTheDocument();
});
