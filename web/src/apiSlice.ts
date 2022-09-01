import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import { ViewInterface, viewOrderInterface } from "features/forwards/forwardsSlice";
import { settings, timeZone, localNode } from "./apiTypes";
import { SortByOptionType } from "./features/sidebar/sections/sort/SortSectionOld";

const buildBaseUrl = () => {
  // checks to see if the app is running under /torq and if so prepends that to API paths
  const splitLocation = window.location.pathname.split("/");
  let prefix = "";
  if (splitLocation.length > 1) {
    const path = splitLocation[1];
    if (path === "torq") {
      prefix = "/torq";
    }
  }
  return window.location.port === "3000"
    ? "//" + window.location.hostname + ":8080" + prefix + "/api"
    : "//" + window.location.host + prefix + "/api";
};

const API_URL = buildBaseUrl();

// Define a service using a base URL and expected endpoints
export const torqApi = createApi({
  reducerPath: "api",
  baseQuery: fetchBaseQuery({
    baseUrl: API_URL,
    // prepareHeaders: (headers, _) => {
    //   if (!headers.get("Content-Type")) {
    //     headers.set("Content-Type", "application/json");
    //   }
    //   return headers;
    // },
    credentials: "include",
    mode: "cors",
  }),
  tagTypes: ["settings", "tableView", "localNodes"],
  endpoints: (builder) => ({
    getFlow: builder.query<any, { from: string; to: string; chanId: string }>({
      query: ({ from, to, chanId }) => `flow?from=${from}&to=${to}&chan_id=${chanId}`,
    }),
    getChannelHistory: builder.query<any, { from: string; to: string; chanIds: string }>({
      query: ({ from, to, chanIds }) => `channels/${chanIds}?from=${from}&to=${to}`,
    }),
    getForwards: builder.query<any, { from: string; to: string }>({
      query: ({ from, to }) => `forwards?from=${from}&to=${to}`,
    }),
    getPayments: builder.query<
      any,
      { limit: number; offset: number; order?: Array<{ key: string; direction: "asc" | "desc" }>; filter?: any }
    >({
      query: ({ limit, offset, order, filter }) =>
        `payments?limit=${limit || 100}&offset=${offset || 0}${order ? "&order=" + JSON.stringify(order) : ""}${
          filter ? "&filter=" + JSON.stringify(filter) : ""
        }`,
    }),
    getInvoices: builder.query<
      any,
      { limit: number; offset: number; order?: Array<{ key: string; direction: "asc" | "desc" }>; filter?: any }
    >({
      query: ({ limit, offset, order, filter }) =>
        `invoices?limit=${limit || 100}&offset=${offset || 0}${order ? "&order=" + JSON.stringify(order) : ""}${
          filter ? "&filter=" + JSON.stringify(filter) : ""
        }`,
    }),
    getOnChainTx: builder.query<
      any,
      { limit: number; offset: number; order?: Array<{ key: string; direction: "asc" | "desc" }>; filter?: any }
    >({
      query: ({ limit, offset, order, filter }) =>
        `on-chain-tx?limit=${limit || 100}&offset=${offset || 0}${order ? "&order=" + JSON.stringify(order) : ""}${
          filter ? "&filter=" + JSON.stringify(filter) : ""
        }`,
    }),
    getTableViews: builder.query<any, void>({
      query: () => `table-views`,
      providesTags: ["tableView"],
    }),
    createTableView: builder.mutation<any, { view: ViewInterface; index: number }>({
      query: (data) => ({
        url: "table-views",
        method: "POST",
        body: { id: null, view: data.view },
      }),
      transformResponse: (response: { view: ViewInterface }, _, arg) => ({
        view: response,
        index: arg.index,
      }),
    }),
    updateTableView: builder.mutation<any, ViewInterface>({
      query: (view: ViewInterface) => ({
        url: "table-views",
        method: "PUT",
        body: { id: view.id, view: view },
      }),
    }),
    deleteTableView: builder.mutation<any, { view: ViewInterface; index: number }>({
      query: (data) => ({
        url: `table-views/${data.view.id}`,
        method: "DELETE",
      }),
      transformResponse: (_, __, arg) => ({ index: arg.index }),
    }),
    updateTableViewsOrder: builder.mutation<any, viewOrderInterface[]>({
      query: (order: viewOrderInterface[]) => ({
        url: "table-views/order",
        method: "PATCH",
        body: order,
      }),
    }),
    logout: builder.mutation<any, void>({
      query: () => ({
        url: "logout",
        method: "POST",
      }),
      invalidatesTags: ["tableView"],
    }),
    login: builder.mutation<any, FormData>({
      query: (form) => ({
        url: "login",
        method: "POST",
        body: new URLSearchParams(form as any),
        // headers: { "Content-Type": "application/x-www-form-urlencoded" },
      }),
    }),
    getSettings: builder.query<settings, void>({
      query: () => `settings`,
      providesTags: ["settings"],
    }),
    updateSettings: builder.mutation<any, settings>({
      query: (settings) => ({
        url: "settings",
        method: "PUT",
        body: settings,
      }),
      invalidatesTags: ["settings"],
    }),
    getTimeZones: builder.query<timeZone[], void>({
      query: () => `settings/timezones`,
    }),
    getLocalNodes: builder.query<localNode[], void>({
      query: () => `settings/local-nodes`,
      providesTags: ["localNodes"],
    }),
    getLocalNode: builder.query<localNode, number>({
      query: (nodeId) => `settings/local-nodes/${nodeId}`,
      providesTags: ["localNodes"],
    }),
    addLocalNode: builder.mutation<any, FormData>({
      query: (localNode) => ({
        url: "settings/local-nodes",
        method: "POST",
        body: localNode,
      }),
      invalidatesTags: ["localNodes"],
    }),
    updateLocalNode: builder.mutation<any, { form: FormData; localNodeId: number }>({
      query: (localNode) => ({
        url: `settings/local-nodes/${localNode.localNodeId}`,
        method: "PUT",
        body: localNode.form,
      }),
      invalidatesTags: ["localNodes"],
    }),
    updateLocalNodeSetDisabled: builder.mutation<any, { localNodeId: number; disabled: boolean }>({
      query: (localNode) => ({
        url: `settings/local-nodes/${localNode.localNodeId}/set-disabled`,
        method: "PUT",
        body: localNode,
      }),
      invalidatesTags: ["localNodes"],
    }),
    updateLocalNodeSetDeleted: builder.mutation<any, { localNodeId: number }>({
      query: (localNode) => ({
        url: `settings/local-nodes/${localNode.localNodeId}`,
        method: "DELETE",
      }),
      invalidatesTags: ["localNodes"],
    }),
  }),
});

// Export hooks for usage in functional components, which are
// auto-generated based on the defined endpoints
export const {
  useGetFlowQuery,
  useGetChannelHistoryQuery,
  useGetForwardsQuery,
  useGetPaymentsQuery,
  useGetInvoicesQuery,
  useGetOnChainTxQuery,
  useGetTableViewsQuery,
  useUpdateTableViewMutation,
  useCreateTableViewMutation,
  useDeleteTableViewMutation,
  useUpdateTableViewsOrderMutation,
  useLoginMutation,
  useLogoutMutation,
  useGetSettingsQuery,
  useUpdateSettingsMutation,
  useGetTimeZonesQuery,
  useGetLocalNodeQuery,
  useGetLocalNodesQuery,
  useUpdateLocalNodeMutation,
  useAddLocalNodeMutation,
  useUpdateLocalNodeSetDeletedMutation,
  useUpdateLocalNodeSetDisabledMutation,
} = torqApi;
