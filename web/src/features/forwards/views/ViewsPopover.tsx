import styles from "./views.module.scss";
import { DragDropContext, Droppable, Draggable } from "react-beautiful-dnd";
import {
  Dismiss20Regular as RemoveIcon,
  Edit16Regular as EditIcon,
  Save20Regular as SaveIcon,
  AddSquare20Regular as AddIcon,
  Reorder20Regular as DragHandle,
} from "@fluentui/react-icons";
import { useState } from "react";
import classNames from "classnames";
import TabButton from "features/buttons/TabButton";
import { useAppDispatch, useAppSelector } from "store/hooks";
import {
  selectViews,
  updateViews,
  updateSelectedView,
  selectedViewIndex,
  ViewInterface,
  viewOrderInterface,
  DefaultView,
  updateViewsOrder,
} from "features/forwards/forwardsSlice";
import Popover from "features/popover/Popover";
import Button, { buttonColor } from "features/buttons/Button";
import {
  useCreateTableViewMutation,
  useUpdateTableViewMutation,
  useDeleteTableViewMutation,
  useUpdateTableViewsOrderMutation,
} from "apiSlice";

interface viewRow {
  title: string;
  index: number;
  handleUpdateView: (view: ViewInterface, index: number) => void;
  handleRemoveView: (index: number) => void;
  handleSelectView: (index: number) => void;
  singleView: boolean;
}

function ViewRow({ title, index, handleUpdateView, handleRemoveView, handleSelectView, singleView }: viewRow) {
  const [editView, setEditView] = useState(false);
  const [localTitle, setLocalTitle] = useState(title);

  function handleInputChange(e: any) {
    setLocalTitle(e.target.value);
  }

  function handleInputSubmit(e: any) {
    e.preventDefault();
    setEditView(false);
    handleUpdateView({ title: localTitle } as ViewInterface, index);
  }

  return (
    <Draggable draggableId={`draggable-view-id-${index}`} index={index}>
      {(provided, snapshot) => (
        <div
          className={classNames(styles.viewRow, { dragging: snapshot.isDragging })}
          ref={provided.innerRef}
          {...provided.draggableProps}
        >
          <div className={styles.viewRowDragHandle} {...provided.dragHandleProps}>
            <DragHandle />
          </div>

          {editView ? (
            <form onSubmit={handleInputSubmit} className={classNames(styles.viewEdit, "torq-input-field")}>
              <input type="text" autoFocus={true} onChange={handleInputChange} value={localTitle} />
              <button type={"submit"}>
                <SaveIcon />
              </button>
            </form>
          ) : (
            <div className={styles.viewSelect} onClick={() => handleSelectView(index)}>
              <div>{title}</div>
              <div className={styles.editView} onClick={() => setEditView(true)}>
                <EditIcon />
              </div>
            </div>
          )}
          {!singleView ? (
            <div className={styles.removeView} onClick={() => handleRemoveView(index)}>
              <RemoveIcon />
            </div>
          ) : (
            <div className={classNames(styles.removeView, styles.disabled)}>
              <RemoveIcon />
            </div>
          )}
        </div>
      )}
    </Draggable>
  );
}

function ViewsPopover() {
  const views = useAppSelector(selectViews);
  const selectedView = useAppSelector(selectedViewIndex);
  const dispatch = useAppDispatch();
  const [updateTableView] = useUpdateTableViewMutation();
  const [deleteTableView] = useDeleteTableViewMutation();
  const [createTableView] = useCreateTableViewMutation();
  const [updateTableViewsOrder] = useUpdateTableViewsOrderMutation();

  const updateView = (view: ViewInterface, index: number) => {
    const updatedViews = [
      ...views.slice(0, index),
      { ...views[index], ...view },
      ...views.slice(index + 1, views.length),
    ];
    dispatch(updateViews({ views: updatedViews, index: index }));
    updateTableView({ ...views[index], ...view });
  };

  const removeView = (index: number) => {
    const confirmed = window.confirm("Are you sure you want to delete this view?");
    if (!confirmed) {
      return;
    }
    deleteTableView({ view: views[index], index: index });
  };

  const addView = () => {
    createTableView({ view: DefaultView, index: views.length });
  };

  const selectView = (index: number) => {
    dispatch(updateSelectedView({ index: index }));
  };

  const droppableContainerId = "views-list-droppable";

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

    let newCurrentIndex = selectedView;
    // If moved view is the selected view
    if (source.index === selectedView) {
      newCurrentIndex = destination.index;
    }
    // If moved view has a source bellow the selected view
    if (source.index < selectedView && destination.index >= selectedView) {
      newCurrentIndex = selectedView - 1;
    }
    // If moved view has a source above the selected view
    if (source.index > selectedView && destination.index <= selectedView) {
      newCurrentIndex = selectedView + 1;
    }

    const newViewsOrder: ViewInterface[] = views.slice();
    newViewsOrder.splice(source.index, 1);
    newViewsOrder.splice(destination.index, 0, views[source.index]);

    dispatch(updateViewsOrder({ views: newViewsOrder, index: newCurrentIndex }));

    const order: viewOrderInterface[] = newViewsOrder.map((view, index) => {
      return { id: view.id, view_order: index };
    });
    updateTableViewsOrder(order);
  };

  const button = <TabButton title={views[selectedView].title} dropDown={true} />;

  const singleView = views.length <= 1;
  return (
    <Popover button={button}>
      <DragDropContext onDragEnd={onDragEnd}>
        <div className={styles.viewsPopoverContent}>
          <Droppable droppableId={droppableContainerId}>
            {(provided) => (
              <div className={styles.viewRows} ref={provided.innerRef} {...provided.droppableProps}>
                {views.map((view, index) => {
                  return (
                    <ViewRow
                      title={view.title}
                      key={index}
                      index={index}
                      handleRemoveView={removeView}
                      handleUpdateView={updateView}
                      handleSelectView={selectView}
                      singleView={singleView}
                    />
                  );
                })}
                {provided.placeholder}
              </div>
            )}
          </Droppable>
          <div className={styles.buttonsRow}>
            <Button buttonColor={buttonColor.ghost} text={"Add View"} icon={<AddIcon />} onClick={addView} />
          </div>
        </div>
      </DragDropContext>
    </Popover>
  );
}

export default ViewsPopover;
