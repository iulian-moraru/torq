import { createSlice, PayloadAction } from "@reduxjs/toolkit";
import { RootState } from "store/store";
import {
  AndClause,
  FilterClause
} from "features/sidebar/sections/filter/filter";
import { SortByOptionType } from "features/sidebar/sections/sort/SortSectionOld";
import { torqApi } from "apiSlice";
import { ColumnMetaData, ViewInterface } from "features/table/Table";

export const availableColumns: ColumnMetaData[] = [
  {
    heading: "Name",
    type: "AliasCell",
    key: "alias",
    locked: true,
    valueType: "string"
  },
  {
    heading: "Revenue",
    type: "BarCell",
    key: "revenueOut",
    valueType: "number"
  },
  {
    heading: "Total Forwards",
    type: "BarCell",
    key: "countTotal",
    valueType: "number"
  },
  {
    heading: "Outbound Amount",
    type: "BarCell",
    key: "amountOut",
    valueType: "number"
  },
  {
    heading: "Inbound Amount",
    type: "BarCell",
    key: "amountIn",
    valueType: "number"
  },
  {
    heading: "Total Amount",
    type: "BarCell",
    key: "amountTotal",
    valueType: "number"
  },
  {
    heading: "Turnover Outbound",
    type: "BarCell",
    key: "turnoverOut",
    valueType: "number"
  },
  {
    heading: "Turnover Inbound",
    type: "BarCell",
    key: "turnoverIn",
    valueType: "number"
  },
  {
    heading: "Total Turnover",
    type: "BarCell",
    key: "turnoverTotal",
    valueType: "number"
  },
  {
    heading: "Outbound Forwards",
    type: "BarCell",
    key: "countOut",
    valueType: "number"
  },
  {
    heading: "Inbound Forwards",
    type: "BarCell",
    key: "countIn",
    valueType: "number"
  },
  {
    heading: "Revenue inbound",
    type: "BarCell",
    key: "revenueIn",
    valueType: "number"
  },
  {
    heading: "Revenue total",
    type: "BarCell",
    key: "revenueTotal",
    valueType: "number"
  },
  {
    heading: "Capacity",
    type: "BarCell",
    key: "capacity",
    valueType: "number"
  },
  {
    heading: "Public key",
    type: "TextCell",
    key: "pubKey",
    valueType: "string"
  },
  {
    heading: "Funding Transaction",
    type: "TextCell",
    key: "fundingTransactionHash",
    valueType: "string"
  },
  {
    heading: "Channel short ID",
    type: "TextCell",
    key: "shortChannelId",
    valueType: "string"
  },
  {
    heading: "LND Channel short ID",
    type: "TextCell",
    key: "lndShortChannelId",
    valueType: "string"
  },
  {
    heading: "Open",
    type: "BooleanCell",
    key: "open",
    valueType: "boolean"
  }
];

export interface TableState {
  channels: [];
  modChannels: [];
  selectedViewIndex: number;
  views: ViewInterface[];
  status: "idle" | "loading" | "failed";
}

const defaultFilter = new AndClause();
defaultFilter.addChildClause(
  new FilterClause({
    funcName: "gt",
    category: "number" as "number" | "string",
    key: "amountTotal",
    parameter: 0
  })
);

export const activeForwardsColumns = availableColumns.filter((c) =>
  [
    "alias",
    "revenueOut",
    "countTotal",
    "amountOut",
    "amountIn",
    "amountTotal",
    "turnoverTotal",
    "capacity"
  ].includes(c.key)
);

export const DefaultView: ViewInterface = {
  title: "Untitled View",
  saved: true,
  columns: activeForwardsColumns,
  filters: defaultFilter.toJSON(),
  sortBy: [{ value: "revenueOut", label: "Revenue", direction: "desc" }],
  groupBy: undefined,
  page: "forwards"
};

const initialState: TableState = {
  channels: [],
  modChannels: [],
  selectedViewIndex: 0,
  views: [
    {
      ...DefaultView,
      title: "Default View"
    }
  ],
  status: "idle"
};

export const forwardsSlice = createSlice({
  name: "table",
  initialState,
  // The `reducers` field lets us define reducers and generate associated actions
  reducers: {
    updateFilters: (state, actions: PayloadAction<{ filters: any }>) => {
      state.views[state.selectedViewIndex].filters = actions.payload.filters;
    },
    updateColumns: (
      state,
      actions: PayloadAction<{ columns: ColumnMetaData[] }>
    ) => {
      state.views[state.selectedViewIndex].columns = actions.payload.columns;
    },
    updateGroupBy: (state, actions: PayloadAction<{ groupBy: string }>) => {
      state.views[state.selectedViewIndex].groupBy = actions.payload.groupBy;
    },
    updateViews: (
      state,
      actions: PayloadAction<{ views: ViewInterface[]; index: number }>
    ) => {
      state.views = actions.payload.views;
      state.selectedViewIndex = actions.payload.index;
    },
    updateViewsOrder: (
      state,
      actions: PayloadAction<{ views: ViewInterface[]; index: number }>
    ) => {
      state.views = actions.payload.views;
      state.selectedViewIndex = actions.payload.index;
    },
    deleteView: (
      state,
      actions: PayloadAction<{ view: ViewInterface; index: number }>
    ) => {
      state.views = [
        ...state.views.slice(0, actions.payload.index),
        ...state.views.slice(actions.payload.index + 1, state.views.length)
      ];
      state.selectedViewIndex = 0;
    },
    updateSelectedView: (state, actions: PayloadAction<{ index: number }>) => {
      state.selectedViewIndex = actions.payload.index;
    },
    updateSortBy: (
      state,
      actions: PayloadAction<{ sortBy: SortByOptionType[] }>
    ) => {
      state.views[state.selectedViewIndex].sortBy = actions.payload.sortBy;
    }
  },
  // The `extraReducers` field lets the slice handle actions defined elsewhere,
  // including actions generated by createAsyncThunk or in other slices.
  extraReducers: (builder) => {
    builder.addMatcher(
      (action) => {
        return (
          [
            "table/updateFilters",
            "table/updateSortBy",
            "table/updateColumns",
            "table/updateGroupBy"
          ].findIndex((item) => action.type === item) !== -1
        );
      },
      (state, _) => {
        // TODO: create compare version to indicate it view is saved or not.
        state.views[state.selectedViewIndex].saved = false;
      }
    );

    builder.addMatcher(
      torqApi.endpoints.createTableView.matchFulfilled,
      (state, { payload }) => {
        state.views[payload.index] = {
          ...payload.view.view,
          id: payload.view.id
        };
        state.selectedViewIndex = payload.index;
      }
    );

    builder.addMatcher(
      torqApi.endpoints.deleteTableView.matchFulfilled,
      (state, { payload }) => {
        state.views = [
          ...state.views.slice(0, payload.index),
          ...state.views.slice(payload.index + 1, state.views.length)
        ];
        state.selectedViewIndex = 0;
      }
    );

    builder.addMatcher(
      torqApi.endpoints.getTableViews.matchFulfilled,
      (state, { payload }) => {
        if (payload !== null) {
          state.views = payload.map(
            (view: { id: number; view: ViewInterface }) => {
              return { ...view.view, id: view.id };
            }
          );
        }
      }
    );

    builder.addMatcher(
      torqApi.endpoints.updateTableView.matchFulfilled,
      (state, { payload }) => {
        const view = state.views.find((v) => v.id === payload.id);
        if (view) {
          view.saved = true;
        }
      }
    );
  }
});

export const {
  updateFilters,
  updateViews,
  updateViewsOrder,
  deleteView,
  updateSelectedView,
  updateSortBy,
  updateColumns,
  updateGroupBy
} = forwardsSlice.actions;

export const selectActiveColumns = (state: RootState) => {
  return state.table.views[state.table.selectedViewIndex].columns || [];
};
export const selectAllColumns = (_: RootState) => availableColumns;
export const selectSortBy = (state: RootState) =>
  state.table.views[state.table.selectedViewIndex].sortBy;
export const selectGroupBy = (state: RootState) =>
  state.table.views[state.table.selectedViewIndex].groupBy;
export const selectFilters = (state: RootState) => {
  return state.table.views[state.table.selectedViewIndex].filters;
};
export const selectViews = (state: RootState) => state.table.views;
export const selectCurrentView = (state: RootState) =>
  state.table.views[state.table.selectedViewIndex];
export const selectedViewIndex = (state: RootState) =>
  state.table.selectedViewIndex;
export const selectStatus = (state: RootState) => state.table.status;

export default forwardsSlice.reducer;
