import TabButton from "features/buttons/TabButton";
import { TableControlsTabsGroup } from "features/templates/tablePageTemplate/TablePageTemplate";

function TransactTabs() {
  return (
    <TableControlsTabsGroup>
      {/*<TabButton to={"/transactions/all"} title={"All"} />*/}
      <TabButton to={"/transactions/payments"} title={"Payments"} />
      <TabButton to={"/transactions/invoices"} title={"Invoices"} />
      <TabButton to={"/transactions/onchain"} title={"On-Chain"} />
    </TableControlsTabsGroup>
  );
}

export default TransactTabs;
