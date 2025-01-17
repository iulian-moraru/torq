import Table, { ColumnMetaData } from "features/table/Table";
import { useCreateTableViewMutation, useGetTableViewsQuery, useUpdateTableViewMutation, useGetOnChainTxQuery } from "apiSlice";
import { Link, useNavigate } from "react-router-dom";
import {
  Filter20Regular as FilterIcon,
  ArrowSortDownLines20Regular as SortIcon,
  ColumnTriple20Regular as ColumnsIcon,
  Options20Regular as OptionsIcon,
  LinkEdit20Regular as NewOnChainAddressIcon,
  Save20Regular as SaveIcon,
} from "@fluentui/react-icons";
import Sidebar from "features/sidebar/Sidebar";
import TablePageTemplate, {
  TableControlSection,
  TableControlsButton,
  TableControlsButtonGroup,
} from "features/templates/tablePageTemplate/TablePageTemplate";
import { useEffect, useState } from "react";
import TransactTabs from "features/transact/TransactTabs";
import Pagination from "features/table/pagination/Pagination";
import useLocalStorage from "features/helpers/useLocalStorage";
import SortSection, { OrderBy } from "features/sidebar/sections/sort/SortSection";
import FilterSection from "features/sidebar/sections/filter/FilterSection";
import { Clause, FilterInterface } from "features/sidebar/sections/filter/filter";
import { useAppDispatch, useAppSelector } from "store/hooks";
import {
  selectViews,
  updateViews,
  updateSelectedView,
  updateViewsOrder,
  DefaultView,
  selectAllColumns,
  selectOnChainFilters,
  updateColumns,
  updateOnChainFilters,
  selectCurrentView,
  selectedViewIndex,
  selectActiveColumns,
} from "features/transact/OnChain/onChainSlice";
import { FilterCategoryType } from "features/sidebar/sections/filter/filter";
import ColumnsSection from "features/sidebar/sections/columns/ColumnsSection";
import { SectionContainer } from "features/section/SectionContainer";
import Button, { buttonColor } from "features/buttons/Button";
import { NEW_ADDRESS } from "constants/routes";
import { useLocation } from "react-router";
import useTranslations from "services/i18n/useTranslations";
import { ViewInterface } from "features/table/Table";
import { ViewResponse } from "features/viewManagement/ViewsPopover";
type sections = {
  filter: boolean;
  sort: boolean;
  columns: boolean;
};

const statusTypes: any = {
  OPEN: "Open",
  SETTLED: "Settled",
  EXPIRED: "Expired",
};
function OnChainPage() {
  const dispatch = useAppDispatch();

  const { data: onchainViews, isLoading } = useGetTableViewsQuery({page: 'onChain'});

  useEffect(() => {
    const views: ViewInterface[] = [];
    if (!isLoading) {
      if (onchainViews) {
        onchainViews?.map((v: ViewResponse) => {
          views.push(v.view)
        });

        dispatch(updateViews({ views, index: 0 }));
      } else {
        dispatch(updateViews({ views: [{...DefaultView, title: "Default View"}], index: 0 }));
      }
    }
  }, [onchainViews, isLoading]);

  const [limit, setLimit] = useLocalStorage("onchainLimit", 100);
  const [offset, setOffset] = useState(0);
  const [orderBy, setOrderBy] = useLocalStorage("onchainOrderBy", [
    {
      key: "date",
      direction: "desc",
    },
  ] as OrderBy[]);

  const activeColumns = useAppSelector(selectActiveColumns) || [];
  const allColumns = useAppSelector(selectAllColumns);

  const navigate = useNavigate();
  const filters = useAppSelector(selectOnChainFilters);

  const onchainResponse = useGetOnChainTxQuery({
    limit: limit,
    offset: offset,
    order: orderBy,
  });

  // Logic for toggling the sidebar
  const [sidebarExpanded, setSidebarExpanded] = useState(false);
  let data: any = [];

  if (onchainResponse?.data?.data) {
    data = onchainResponse?.data?.data.map((invoice: any) => {
      const invoice_state = statusTypes[invoice.invoice_state];

      return {
        ...invoice,
        invoice_state,
      };
    });
  }

  // General logic for toggling the sidebar sections
  const initialSectionState: sections = {
    filter: false,
    sort: false,
    columns: false,
  };

  const [activeSidebarSections, setActiveSidebarSections] = useState(initialSectionState);

  const sidebarSectionHandler = (section: keyof sections) => {
    return () => {
      setActiveSidebarSections({
        ...activeSidebarSections,
        [section]: !activeSidebarSections[section],
      });
    };
  };

  const closeSidebarHandler = () => {
    return () => {
      setSidebarExpanded(false);
    };
  };

  const location = useLocation();
  const { t } = useTranslations();

  const [updateTableView] = useUpdateTableViewMutation();
  const [createTableView] = useCreateTableViewMutation();
  const currentViewIndex = useAppSelector(selectedViewIndex);
  const currentView = useAppSelector(selectCurrentView);
  const saveView = () => {
    const viewMod = { ...currentView };
    viewMod.saved = true;
    if (currentView.id === undefined || null) {
      createTableView({ view: viewMod, index: currentViewIndex, page: 'onChain' });
      return;
    }
    updateTableView(viewMod);
  };

  const tableControls = (
    <TableControlSection>
      <TransactTabs
        page="onChain"
        selectViews={selectViews}
        updateViews={updateViews}
        updateSelectedView={updateSelectedView}
        selectedViewIndex={selectedViewIndex}
        updateViewsOrder={updateViewsOrder}
        DefaultView={DefaultView}
      />
      {!currentView.saved && (
        <Button
          buttonColor={buttonColor.green}
          icon={<SaveIcon />}
          text={"Save"}
          onClick={saveView}
          className={"collapse-tablet"}
        />
      )}
      <TableControlsButtonGroup>
        <Button
          buttonColor={buttonColor.green}
          text={t.newAddress}
          icon={<NewOnChainAddressIcon />}
          className={"collapse-tablet"}
          onClick={() => {
            navigate(NEW_ADDRESS, { state: { background: location } });
          }}
        />
        <TableControlsButton
          onClickHandler={() => setSidebarExpanded(!sidebarExpanded)}
          icon={OptionsIcon}
          id={"tableControlsButton"}
        />
      </TableControlsButtonGroup>
    </TableControlSection>
  );

  const defaultFilter: FilterInterface = {
    funcName: "gte",
    category: "number" as FilterCategoryType,
    parameter: 0,
    key: "amount",
  };
  const filterColumns = useAppSelector(selectAllColumns);

  const handleFilterUpdate = (updated: Clause) => {
    dispatch(updateOnChainFilters({ filters: updated.toJSON() }));
  };

  const sortableColumns = allColumns.filter((column: ColumnMetaData) =>
    [
      "date",
      "destAddresses",
      "destAddressesCount",
      "amount",
      "totalFees",
      "label",
      "lndTxTypeLabel",
      "lndShortChanId",
    ].includes(column.key)
  );

  const handleSortUpdate = (updated: Array<OrderBy>) => {
    setOrderBy(updated);
  };

  const updateColumnsHandler = (columns: Array<any>) => {
    dispatch(updateColumns({ columns: columns }));
  };

  const sidebar = (
    <Sidebar title={"Options"} closeSidebarHandler={closeSidebarHandler()}>
      <SectionContainer
        title={"Columns"}
        icon={ColumnsIcon}
        expanded={activeSidebarSections.columns}
        handleToggle={sidebarSectionHandler("columns")}
      >
        <ColumnsSection columns={allColumns} activeColumns={activeColumns} handleUpdateColumn={updateColumnsHandler} />
      </SectionContainer>
      <SectionContainer
        title={"Filter"}
        icon={FilterIcon}
        expanded={activeSidebarSections.filter}
        handleToggle={sidebarSectionHandler("filter")}
      >
        <FilterSection
          columnsMeta={filterColumns}
          filters={filters}
          filterUpdateHandler={handleFilterUpdate}
          defaultFilter={defaultFilter}
        />
      </SectionContainer>
      <SectionContainer
        title={"Sort"}
        icon={SortIcon}
        expanded={activeSidebarSections.sort}
        handleToggle={sidebarSectionHandler("sort")}
      >
        <SortSection columns={sortableColumns} orderBy={orderBy} updateHandler={handleSortUpdate} />
      </SectionContainer>
    </Sidebar>
  );

  const breadcrumbs = [
    <span key="b1">Transactions</span>,
    <Link key="b2" to={"/transactions/on-chain"}>
      On-Chain Tx
    </Link>,
  ];
  const pagination = (
    <Pagination
      limit={limit}
      offset={offset}
      total={onchainResponse?.data?.pagination?.total || 0}
      perPageHandler={setLimit}
      offsetHandler={setOffset}
    />
  );
  return (
    <TablePageTemplate
      title={"OnChain"}
      breadcrumbs={breadcrumbs}
      sidebarExpanded={sidebarExpanded}
      sidebar={sidebar}
      tableControls={tableControls}
      pagination={pagination}
    >
      <Table
        data={data}
        activeColumns={activeColumns || []}
        isLoading={onchainResponse.isLoading || onchainResponse.isFetching || onchainResponse.isUninitialized}
      />
    </TablePageTemplate>
  );
}

export default OnChainPage;
