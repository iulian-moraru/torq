import { createSlice, PayloadAction } from "@reduxjs/toolkit";
import { ColumnMetaData } from "features/table/Table";
import { RootState } from "store/store";
import { torqApi } from "apiSlice";
import { ViewInterface } from "features/table/Table";

export interface TableInvoiceState {
  invoiceViews: ViewInterface[];
  selectedViewIndex: number;
}

export const AllInvoicesColumns: ColumnMetaData[] = [
  {
    key: "creationDate",
    heading: "Creation Date",
    type: "DateCell",
    valueType: "date",
  },
  {
    key: "settleDate",
    heading: "Settle Date",
    type: "DateCell",
    valueType: "date",
  },
  {
    key: "invoiceState",
    heading: "State",
    type: "TextCell",
    valueType: "array",
  },
  {
    key: "amtPaid",
    heading: "Paid Amount",
    type: "NumericCell",
    valueType: "number",
  },
  {
    key: "memo",
    heading: "memo",
    type: "TextCell",
    valueType: "string",
  },
  {
    key: "value",
    heading: "Invoice Amount",
    type: "NumericCell",
    valueType: "number",
  },
  {
    key: "isRebalance",
    heading: "Rebalance",
    type: "BooleanCell",
    valueType: "boolean",
  },
  {
    key: "isKeysend",
    heading: "Keysend",
    type: "BooleanCell",
    valueType: "boolean",
  },
  {
    key: "destinationPubKey",
    heading: "Destination",
    type: "TextCell",
    valueType: "string",
  },
  {
    key: "isAmp",
    heading: "AMP",
    type: "BooleanCell",
    valueType: "boolean",
  },
  {
    key: "fallbackAddr",
    heading: "Fallback Address",
    type: "TextCell",
    valueType: "string",
  },
  {
    key: "paymentAddr",
    heading: "Payment Address",
    type: "TextCell",
    valueType: "string",
  },
  {
    key: "paymentRequest",
    heading: "Payment Request",
    type: "TextCell",
    valueType: "string",
  },
  {
    key: "private",
    heading: "Private",
    type: "BooleanCell",
    valueType: "boolean",
  },
  {
    key: "rHash",
    heading: "Hash",
    type: "TextCell",
    valueType: "string",
  },
  {
    key: "rPreimage",
    heading: "Preimage",
    type: "TextCell",
    valueType: "string",
  },
  {
    key: "expiry",
    heading: "Expiry",
    type: "NumericCell",
    valueType: "number",
  },
  {
    key: "cltvExpiry",
    heading: "CLTV Expiry",
    type: "NumericCell",
    valueType: "number",
  },
  {
    key: "updatedOn",
    heading: "Updated On",
    type: "DateCell",
    valueType: "date",
  }
];

export const ActiveInvoicesColumns = AllInvoicesColumns.filter(({ key }) =>
  [
    "creationDate",
    "settleDate",
    "invoiceState",
    "amtPaid",
    "memo",
    "value",
    "isRebalance",
    "isKeysend",
    "destinationPubKey",
  ].includes(key)
);

export const DefaultView: ViewInterface = {
  title: "Untitled View",
  saved: true,
  columns: ActiveInvoicesColumns,
  page: "invoices",
  sortBy: [],
};

const initialState: TableInvoiceState = {
  selectedViewIndex: 0,
  invoiceViews: [
    {
      ...DefaultView,
      title: "Default View",
    },
  ],
};

export const invoicesSlice = createSlice({
  name: "invoices",
  initialState,
  reducers: {
    updateInvoicesFilters: (state, actions: PayloadAction<{ filters: any }>) => {
      state.invoiceViews[0].filters = actions.payload.filters;
    },
    updateColumns: (state, actions: PayloadAction<{ columns: ColumnMetaData[] }>) => {
      state.invoiceViews[0].columns = actions.payload.columns;
    },
    updateViews: (state, actions: PayloadAction<{ views: ViewInterface[]; index: number }>) => {
      state.invoiceViews = actions.payload.views;
      state.selectedViewIndex = actions.payload.index;
    },
    updateViewsOrder: (state, actions: PayloadAction<{ views: ViewInterface[]; index: number }>) => {
      state.invoiceViews = actions.payload.views;
      state.selectedViewIndex = actions.payload.index;
    },
    deleteView: (state, actions: PayloadAction<{ view: ViewInterface; index: number }>) => {
      state.invoiceViews = [
        ...state.invoiceViews.slice(0, actions.payload.index),
        ...state.invoiceViews.slice(actions.payload.index + 1, state.invoiceViews.length),
      ];
      state.selectedViewIndex = 0;
    },
    updateSelectedView: (state, actions: PayloadAction<{ index: number }>) => {
      state.selectedViewIndex = actions.payload.index;
    },
  },
  // The `extraReducers` field lets the slice handle actions defined elsewhere,
  // including actions generated by createAsyncThunk or in other slices.
  extraReducers: (builder) => {
    builder.addMatcher(
      (action) => {
        return (
          ["invoices/updateInvoicesFilters", "invoices/updateColumns"].findIndex(
            (item) => action.type === item
          ) !== -1
        );
      },
      (state, _) => {
        state.invoiceViews[state.selectedViewIndex].saved = false;
      }
    );

    builder.addMatcher(torqApi.endpoints.createTableView.matchFulfilled, (state, { payload }) => {
      state.invoiceViews[payload.index] = {
        ...payload.view.view,
        id: payload.view.id,
      };
      state.selectedViewIndex = payload.index;
    });

    builder.addMatcher(torqApi.endpoints.deleteTableView.matchFulfilled, (state, { payload }) => {
      state.invoiceViews = [
        ...state.invoiceViews.slice(0, payload.index),
        ...state.invoiceViews.slice(payload.index + 1, state.invoiceViews.length),
      ];
      state.selectedViewIndex = 0;
    });

    builder.addMatcher(torqApi.endpoints.getTableViews.matchFulfilled, (state, { payload }) => {
      if (payload !== null) {
        state.invoiceViews = payload.map((view: { id: number; view: ViewInterface }) => {
          return { ...view.view, id: view.id };
        });
      }
    });

    builder.addMatcher(torqApi.endpoints.updateTableView.matchFulfilled, (state, { payload }) => {
      const view = state.invoiceViews.find((v) => v.id === payload.id);
      if (view) {
        view.saved = true;
      }
    });
  },
});

export const {
  updateInvoicesFilters,
  updateColumns,
  updateViews,
  updateViewsOrder,
  deleteView,
  updateSelectedView,
} = invoicesSlice.actions;

export const selectInvoicesFilters = (state: RootState) => {
  return state.invoices.invoiceViews[state.invoices.selectedViewIndex].filters;
};

export const selectActiveColumns = (state: RootState) => {
  return state.invoices.invoiceViews[state.invoices.selectedViewIndex].columns || [];
};

export const selectAllColumns = (_: RootState) => AllInvoicesColumns;
export const selectViews = (state: RootState) => state.invoices.invoiceViews;
export const selectCurrentView = (state: RootState) => state.invoices.invoiceViews[state.invoices.selectedViewIndex];
export const selectedViewIndex = (state: RootState) => state.invoices.selectedViewIndex;

export default invoicesSlice.reducer;
