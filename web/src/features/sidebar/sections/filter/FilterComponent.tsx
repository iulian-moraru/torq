import FilterRow from "./FilterRow";
import classNames from "classnames";
import { AddSquare20Regular as AddFilterIcon, AddSquareMultiple20Regular as AddGroupIcon } from "@fluentui/react-icons";

import styles from "./filter-section.module.scss";
import { AndClause, OrClause, Clause, FilterClause, FilterInterface } from "./filter";
import { ColumnMetaData } from "features/table/Table";

interface filterProps {
  columnsMeta: Array<ColumnMetaData>;
  filters: Clause;
  defaultFilter: FilterInterface;
  onFilterUpdate: () => void;
  onNoChildrenLeft?: () => void;
  child: boolean;
}

const combinerOptions = new Map<string, string>([
  ["$and", "And"],
  ["$or", "Or"],
]);

const FilterComponent = (props: filterProps) => {
  const updateFilter = () => {
    props.onFilterUpdate();
  };

  const removeFilter = (index: number) => {
    const filters = props.filters as AndClause | OrClause;
    filters.childClauses.splice(index, 1);
    if (filters.childClauses.length === 1) {
      filters.prefix = "$and";
    }
    if (filters.childClauses.length === 0) {
      if (props.onNoChildrenLeft) {
        props.onNoChildrenLeft();
      }
    }
    props.onFilterUpdate();
  };

  const handleNoChildrenLeft = (index: number) => {
    return () => {
      removeFilter(index);
    };
  };

  const addFilter = () => {
    if (!props.filters) {
      props.filters = new AndClause();
    }
    const filters = props.filters as AndClause | OrClause;
    filters.addChildClause(new FilterClause(props.defaultFilter));
    props.onFilterUpdate();
  };

  const addGroup = () => {
    if (!props.filters) {
      props.filters = new AndClause();
    }
    const filters = props.filters as AndClause | OrClause;
    filters.addChildClause(
      new AndClause([
        new FilterClause({
          funcName: "gte",
          category: "number",
          parameter: 0,
        }),
      ])
    );
    props.onFilterUpdate();
  };

  const handleCombinerChange = () => {
    // this effectively changes the type of the object from AndClause to OrClause or vice versa
    props.filters.prefix = props.filters.prefix === "$and" ? "$or" : "$and";
    props.onFilterUpdate();
  };

  const filterOptions = props.columnsMeta.slice().map((column: any) => {
    const columnOption: any = {
      value: column.key,
      label: column.heading,
      valueType: column.valueType as "string" | "number" | "boolean" | "date" | "array",
      selectOptions: column.selectOptions,
    };
    if (column.valueType === "array") {
      columnOption.arrayOptions = [
        { value: true, label: "True" },
        { value: false, label: "False" },
      ];
    }
    return columnOption;
  });

  filterOptions.sort((a: any, b: any) => {
    if (a.label < b.label) {
      return -1;
    }
    if (a.label > b.label) {
      return 1;
    }
    return 0;
  });

  return (
    <div className={classNames({ [styles.childFilter]: props.child })}>
      <div className={styles.filterRows}>
        {!props.filters.length && <div className={styles.noFilters}>No filters</div>}

        {(props.filters as AndClause | OrClause).childClauses.map((filter, index) => {
          if (filter.prefix === "$filter") {
            return (
              <div
                key={"filter-row-wrapper-" + index}
                className={classNames(styles.filterRow, { [styles.first]: !index })}
              >
                <FilterRow
                  child={props.child}
                  key={"filter-row-" + index}
                  filterClause={filter as FilterClause}
                  index={index}
                  filterOptions={filterOptions}
                  onUpdateFilter={updateFilter}
                  onRemoveFilter={removeFilter}
                  combiner={combinerOptions.get(props.filters.prefix)}
                  handleCombinerChange={handleCombinerChange}
                />
              </div>
            );
          } else {
            return (
              <div
                key={"filter-sub-group-" + index}
                className={classNames(styles.filterRow, { first: !index, [styles.childWrapper]: true })}
              >
                <div className={styles.filterKeyLabel} onClick={() => handleCombinerChange()}>
                  {combinerOptions.get(props.filters.prefix)}
                </div>
                <FilterComponent
                  child={true}
                  defaultFilter={props.defaultFilter}
                  columnsMeta={props.columnsMeta}
                  filters={filter}
                  onNoChildrenLeft={handleNoChildrenLeft(index)}
                  onFilterUpdate={props.onFilterUpdate}
                />
              </div>
            );
          }
        })}
      </div>

      <div className={styles.buttonsRow}>
        <div className={styles.addFilterButton} onClick={addFilter}>
          <AddFilterIcon />
          <span className={styles.buttonText}>{"Add filter"}</span>
        </div>
        {!props.child && (
          <div className={styles.addFilterButton} onClick={addGroup}>
            <AddGroupIcon />
            <span className={styles.buttonText}>{"Add group"}</span>
          </div>
        )}
      </div>
    </div>
  );
};

export default FilterComponent;
