import { createSlice, PayloadAction } from "@reduxjs/toolkit";
import { RootState } from "../../store/store";
import { AndClause, FilterClause } from "../sidebar/sections/filter/filter";
import { SortByOptionType } from "../sidebar/sections/sort/SortSectionOld";
import { torqApi } from "apiSlice";
import { ColumnMetaData } from "../table/Table";

export const availableColumns: ColumnMetaData[] = [
  {
    heading: "Name",
    type: "AliasCell",
    key: "alias",
    locked: true,
    valueType: "string",
  },
  {
    heading: "Revenue",
    type: "BarCell",
    key: "revenue_out",
    valueType: "number",
  },
  {
    heading: "Total Forwards",
    type: "BarCell",
    key: "count_total",
    valueType: "number",
  },
  {
    heading: "Outbound Amount",
    type: "BarCell",
    key: "amount_out",
    valueType: "number",
  },
  {
    heading: "Inbound Amount",
    type: "BarCell",
    key: "amount_in",
    valueType: "number",
  },
  {
    heading: "Total Amount",
    type: "BarCell",
    key: "amount_total",
    valueType: "number",
  },
  {
    heading: "Turnover Outbound",
    type: "BarCell",
    key: "turnover_out",
    valueType: "number",
  },
  {
    heading: "Turnover Inbound",
    type: "BarCell",
    key: "turnover_in",
    valueType: "number",
  },
  {
    heading: "Total Turnover",
    type: "BarCell",
    key: "turnover_total",
    valueType: "number",
  },
  {
    heading: "Outbound Forwards",
    type: "BarCell",
    key: "count_out",
    valueType: "number",
  },
  {
    heading: "Inbound Forwards",
    type: "BarCell",
    key: "count_in",
    valueType: "number",
  },
  {
    heading: "Revenue inbound",
    type: "BarCell",
    key: "revenue_in",
    valueType: "number",
  },
  {
    heading: "Revenue total",
    type: "BarCell",
    key: "revenue_total",
    valueType: "number",
  },
  {
    heading: "Capacity",
    type: "BarCell",
    key: "capacity",
    valueType: "number",
  },
  {
    heading: "Public key",
    type: "TextCell",
    key: "pub_key",
    valueType: "string",
  },
  {
    heading: "Channel point",
    type: "TextCell",
    key: "lndChannelPoint",
    valueType: "string",
  },
  {
    heading: "Channel short ID",
    type: "TextCell",
    key: "shortChannelId",
    valueType: "string",
  },
  {
    heading: "LND Channel short ID",
    type: "TextCell",
    key: "chan_id",
    valueType: "string",
  },
  {
    heading: "Open",
    type: "BooleanCell",
    key: "open",
    valueType: "boolean",
  },
];

export interface ViewInterface {
  title: string;
  id?: number;
  saved: boolean;
  filters?: any;
  columns: ColumnMetaData[];
  sortBy: SortByOptionType[];
  groupBy?: string;
}

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
    key: "amount_total",
    parameter: 0,
  })
);

export const DefaultView: ViewInterface = {
  title: "Untitled View",
  saved: true,
  columns: availableColumns.filter((c) =>
    [
      "alias",
      "revenue_out",
      "count_total",
      "amount_out",
      "amount_in",
      "amount_total",
      "turnover_total",
      "capacity",
    ].includes(c.key)
  ),
  filters: defaultFilter.toJSON(),
  sortBy: [{ value: "revenue_out", label: "Revenue", direction: "desc" }],
  groupBy: undefined,
};

const initialState: TableState = {
  channels: [],
  modChannels: [],
  selectedViewIndex: 0,
  views: [
    {
      ...DefaultView,
      title: "Default View",
    },
  ],
  status: "idle",
};

export interface viewOrderInterface {
  id: number | undefined;
  view_order: number;
}

export const forwardsSlice = createSlice({
  name: "table",
  initialState,
  // The `reducers` field lets us define reducers and generate associated actions
  reducers: {
    updateFilters: (state, actions: PayloadAction<{ filters: any }>) => {
      state.views[state.selectedViewIndex].filters = actions.payload.filters;
    },
    updateColumns: (state, actions: PayloadAction<{ columns: ColumnMetaData[] }>) => {
      state.views[state.selectedViewIndex].columns = actions.payload.columns;
    },
    updateGroupBy: (state, actions: PayloadAction<{ groupBy: string }>) => {
      state.views[state.selectedViewIndex].groupBy = actions.payload.groupBy;
    },
    updateViews: (state, actions: PayloadAction<{ views: ViewInterface[]; index: number }>) => {
      state.views = actions.payload.views;
      state.selectedViewIndex = actions.payload.index;
    },
    updateViewsOrder: (state, actions: PayloadAction<{ views: ViewInterface[]; index: number }>) => {
      state.views = actions.payload.views;
      state.selectedViewIndex = actions.payload.index;
    },
    deleteView: (state, actions: PayloadAction<{ view: ViewInterface; index: number }>) => {
      state.views = [
        ...state.views.slice(0, actions.payload.index),
        ...state.views.slice(actions.payload.index + 1, state.views.length),
      ];
      state.selectedViewIndex = 0;
    },
    updateSelectedView: (state, actions: PayloadAction<{ index: number }>) => {
      state.selectedViewIndex = actions.payload.index;
    },
    updateSortBy: (state, actions: PayloadAction<{ sortBy: SortByOptionType[] }>) => {
      state.views[state.selectedViewIndex].sortBy = actions.payload.sortBy;
    },
  },
  // The `extraReducers` field lets the slice handle actions defined elsewhere,
  // including actions generated by createAsyncThunk or in other slices.
  extraReducers: (builder) => {
    builder.addMatcher(
      (action) => {
        return (
          ["table/updateFilters", "table/updateSortBy", "table/updateColumns", "table/updateGroupBy"].findIndex(
            (item) => action.type === item
          ) !== -1
        );
      },
      (state, _) => {
        // TODO: create compare version to indicate it view is saved or not.
        state.views[state.selectedViewIndex].saved = false;
      }
    );

    builder.addMatcher(torqApi.endpoints.createTableView.matchFulfilled, (state, { payload }) => {
      state.views[payload.index] = {
        ...payload.view.view,
        id: payload.view.id,
      };
      state.selectedViewIndex = payload.index;
    });

    builder.addMatcher(torqApi.endpoints.deleteTableView.matchFulfilled, (state, { payload }) => {
      state.views = [
        ...state.views.slice(0, payload.index),
        ...state.views.slice(payload.index + 1, state.views.length),
      ];
      state.selectedViewIndex = 0;
    });

    builder.addMatcher(torqApi.endpoints.getTableViews.matchFulfilled, (state, { payload }) => {
      if (payload !== null) {
        state.views = payload.map((view: { id: number; view: ViewInterface }) => {
          return { ...view.view, id: view.id };
        });
      }
    });

    builder.addMatcher(torqApi.endpoints.updateTableView.matchFulfilled, (state, { payload }) => {
      const view = state.views.find((v) => v.id === payload.id);
      if (view) {
        view.saved = true;
      }
    });

    // builder.addMatcher(torqApi.endpoints.logout.matchFulfilled, (state) => {
    //   Object.assign(state, initialState);
    // });
  },
});

export const {
  updateFilters,
  updateViews,
  updateViewsOrder,
  deleteView,
  updateSelectedView,
  updateSortBy,
  updateColumns,
  updateGroupBy,
} = forwardsSlice.actions;

export const selectActiveColumns = (state: RootState) => {
  return state.table.views[state.table.selectedViewIndex].columns || [];
};
export const selectAllColumns = (_: RootState) => availableColumns;
export const selectSortBy = (state: RootState) => state.table.views[state.table.selectedViewIndex].sortBy;
export const selectGroupBy = (state: RootState) => state.table.views[state.table.selectedViewIndex].groupBy;
export const selectFilters = (state: RootState) => {
  return state.table.views[state.table.selectedViewIndex].filters;
};
export const selectViews = (state: RootState) => state.table.views;
export const selectCurrentView = (state: RootState) => state.table.views[state.table.selectedViewIndex];
export const selectedViewIndex = (state: RootState) => state.table.selectedViewIndex;
export const selectStatus = (state: RootState) => state.table.status;

export default forwardsSlice.reducer;
