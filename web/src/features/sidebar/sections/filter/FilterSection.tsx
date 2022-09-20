import clone from "clone";
import styles from "./filter-section.module.scss";
import { deserialiseQuery, Clause, AndClause, FilterInterface } from "./filter";
import FilterComponent from "./FilterComponent";
import { ColumnMetaData } from "../../../table/Table";

type FilterSectionProps = {
  columnsMeta: Array<ColumnMetaData>;
  filters: Clause;
  filterUpdateHandler: (filters: Clause) => void;
  defaultFilter: FilterInterface;
};

const FilterSection = (props: FilterSectionProps) => {
  const filtersFromStore = clone<Clause>(props.filters);
  const filters = filtersFromStore ? deserialiseQuery(filtersFromStore) : new AndClause();

  const handleFilterUpdate = () => {
    props.filterUpdateHandler(filters);
  };

  return (
    <div className={styles.filterPopoverContent}>
      <FilterComponent
        columnsMeta={props.columnsMeta}
        filters={filters}
        defaultFilter={props.defaultFilter}
        onFilterUpdate={handleFilterUpdate}
        child={false}
      />
    </div>
  );
};

export default FilterSection;
