import classNames from "classnames";
import { DragDropContext, Droppable, Draggable } from "react-beautiful-dnd";
import {
  Delete16Regular as RemoveIcon,
  Options16Regular as OptionsIcon,
  LineHorizontal1Regular as OptionsExpandedIcon,
  LockClosed16Regular as LockClosedIcon,
  LockOpen16Regular as LockOpenIcon,
  AddCircle16Regular as AddIcon,
  ReOrder16Regular as DragHandle,
} from "@fluentui/react-icons";
import styles from "./columns-section.module.scss";
import Select, { SelectOptionType } from "./ColumnDropDown";
import { ColumnMetaData } from "features/table/Table";
import { useState } from "react";

interface columnRow {
  column: ColumnMetaData;
  index: number;
  handleRemoveColumn: (index: number) => void;
  handleUpdateColumn: (columnMetadata: ColumnMetaData, index: number) => void;
}

const CellOptions: SelectOptionType[] = [
  { label: "Number", value: "NumericCell" },
  { label: "Boolean", value: "BooleanCell" },
  { label: "Array", value: "EnumCell" },
  { label: "Bar", value: "BarCell" },
  { label: "Text", value: "TextCell" },
  { label: "Duration", value: "TextCell" },
];

const NumericCellOptions: SelectOptionType[] = [
  { label: "Number", value: "NumericCell" },
  { label: "Bar", value: "BarCell" },
  { label: "Text", value: "TextCell" },
];

function NameColumnRow({ column, index }: { column: ColumnMetaData; index: number }) {
  return (
    <div className={classNames(styles.columnRow, index)}>
      <div className={classNames(styles.rowLeftIcon, styles.lockBtn)}>
        {column.locked ? <LockClosedIcon /> : <LockOpenIcon />}
      </div>

      <div className={styles.columnName}>
        <div>{column.heading}</div>
      </div>
      {/*<Select value={{ label: "Name", value: "AliasCell" }} isDisabled={true} />*/}
    </div>
  );
}

function ColumnRow({ column, index, handleRemoveColumn, handleUpdateColumn }: columnRow) {
  const selectedOption = CellOptions.filter((option) => {
    if (option.value === column.type) {
      return option;
    }
  })[0];

  const [expanded, setExpanded] = useState(false);

  return (
    <Draggable draggableId={`draggable-column-id-${index}`} index={index}>
      {(provided, snapshot) => (
        <div
          className={classNames(styles.rowContent, {
            dragging: snapshot.isDragging,
            [styles.expanded]: expanded,
          })}
          ref={provided.innerRef}
          {...provided.draggableProps}
        >
          <div className={classNames(styles.columnRow)}>
            <div className={classNames(styles.rowLeftIcon, styles.dragHandle)} {...provided.dragHandleProps}>
              <DragHandle />
            </div>

            <div className={styles.columnName}>
              <div>{column.heading}</div>
            </div>

            <div className={styles.rowOptions} onClick={() => setExpanded(!expanded)}>
              {expanded ? <OptionsExpandedIcon /> : <OptionsIcon />}
            </div>

            <div
              className={styles.removeColumn}
              onClick={() => {
                handleRemoveColumn(index + 1);
              }}
            >
              <RemoveIcon />
            </div>
          </div>
          <div className={styles.rowOptionsContainer}>
            <Select
              isDisabled={["date", "array", "string", "boolean", "enum"].includes(column.valueType)}
              options={NumericCellOptions}
              value={selectedOption}
              onChange={(newValue) => {
                handleUpdateColumn(
                  {
                    ...column,
                    type: (newValue as { value: string; label: string }).value,
                  },
                  index + 1
                );
              }}
            />
          </div>
        </div>
      )}
    </Draggable>
  );
}

interface unselectedColumnRow {
  name: string;
  index: number;
  handleAddColumn: (index: number) => void;
}

function UnselectedColumn({ name, index, handleAddColumn }: unselectedColumnRow) {
  return (
    <div
      className={styles.unselectedColumnRow}
      onClick={() => {
        handleAddColumn(index);
      }}
    >
      <div className={styles.unselectedColumnName}>
        <div>{name}</div>
      </div>

      <div className={styles.addColumn}>
        <AddIcon />
      </div>
    </div>
  );
}

type ColumnsSectionProps = {
  activeColumns: ColumnMetaData[];
  columns: ColumnMetaData[];
  handleUpdateColumn: (updatedColumns: Array<ColumnMetaData>) => void;
};

function ColumnsSection(props: ColumnsSectionProps) {
  const updateColumn = (column: ColumnMetaData, index: number) => {
    const updatedColumns = [
      ...props.activeColumns.slice(0, index),
      column,
      ...props.activeColumns.slice(index + 1, props.columns.length),
    ];
    props.handleUpdateColumn(updatedColumns);
  };

  const removeColumn = (index: number) => {
    const updatedColumns: ColumnMetaData[] = [
      ...props.activeColumns.slice(0, index),
      ...props.activeColumns.slice(index + 1, props.activeColumns.length),
    ];
    props.handleUpdateColumn(updatedColumns);
  };

  const addColumn = (index: number) => {
    const updatedColumns: ColumnMetaData[] = [...props.activeColumns, props.columns[index]];
    props.handleUpdateColumn(updatedColumns);
  };

  const droppableContainerId = "column-list-droppable";
  const draggableColumns = props.activeColumns.slice(1, props.activeColumns.length);

  const onDragEnd = (result: any) => {
    const { destination, source } = result;

    // Dropped outside of container
    if (!destination || destination.droppableId !== droppableContainerId) {
      return;
    }

    // Position not changed
    if (destination.droppableId === source.droppableId && destination.index === source.index) {
      return;
    }

    const newColumns: ColumnMetaData[] = draggableColumns.slice();
    newColumns.splice(source.index, 1);
    newColumns.splice(destination.index, 0, draggableColumns[source.index]);
    props.handleUpdateColumn([props.columns[0], ...newColumns]);
  };

  return (
    <div>
      <DragDropContext onDragEnd={onDragEnd}>
        <div className={styles.columnsSectionContent}>
          <div className={styles.columnRows}>
            <NameColumnRow column={props.activeColumns.slice(0, 1)[0]} key={"selected-0-name"} index={0} />

            <Droppable droppableId={droppableContainerId}>
              {(provided) => (
                <div
                  className={classNames(styles.draggableColumns, styles.columnRows)}
                  ref={provided.innerRef}
                  {...provided.droppableProps}
                >
                  {draggableColumns.map((column, index) => {
                    return (
                      <ColumnRow
                        column={column}
                        key={"selected-" + column.key + "-" + index}
                        index={index}
                        handleRemoveColumn={removeColumn}
                        handleUpdateColumn={updateColumn}
                      />
                    );
                  })}
                  {provided.placeholder}
                </div>
              )}
            </Droppable>

            <div className={styles.divider} />

            <div className={styles.unselectedColumnsWrapper}>
              {props.columns.map((column, index) => {
                if (props.activeColumns.findIndex((ac) => ac.key === column.key) === -1) {
                  return (
                    <UnselectedColumn
                      name={column.heading}
                      key={"unselected-" + index}
                      index={index}
                      handleAddColumn={addColumn}
                    />
                  );
                }
              })}
            </div>
          </div>
        </div>
      </DragDropContext>
    </div>
  );
}

export default ColumnsSection;
