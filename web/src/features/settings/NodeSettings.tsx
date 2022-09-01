import Box from "./Box";
import styles from "./NodeSettings.module.scss";
import Select, { SelectOption } from "../forms/Select";
import React, { useState } from "react";
import {
  Save20Regular as SaveIcon,
  PlugConnected20Regular as ConnectedIcon,
  PlugDisconnected20Regular as DisconnectedIcon,
  ChevronDown20Regular as ChevronIcon,
  MoreCircle20Regular as MoreIcon,
  Delete20Regular as DeleteIcon,
  Pause20Regular as PauseIcon,
  Play20Regular as PlayIcon,
} from "@fluentui/react-icons";
import { toastCategory } from "../toast/Toasts";
import ToastContext from "../toast/context";
import File from "../forms/File";
import TextInput from "features/forms/TextInput";
import {
  useGetLocalNodeQuery,
  useUpdateLocalNodeMutation,
  useAddLocalNodeMutation,
  useUpdateLocalNodeSetDisabledMutation,
  useUpdateLocalNodeSetDeletedMutation,
} from "apiSlice";
import { localNode } from "apiTypes";
import classNames from "classnames";
import Collapse from "features/collapse/Collapse";
import Switch from "features/inputs/Slider/Switch";
import Popover from "features/popover/Popover";
import Button, { buttonVariants } from "features/buttons/Button";
import Modal from "features/modal/Modal";

interface nodeProps {
  localNodeId: number;
  collapsed?: boolean;
  addMode?: boolean;
  onAddSuccess?: Function;
  onAddFailure?: Function;
}
function NodeSettings({ localNodeId, collapsed, addMode, onAddSuccess }: nodeProps) {
  const toastRef = React.useContext(ToastContext);
  const popoverRef = React.useRef();

  const { data: localNodeData } = useGetLocalNodeQuery(localNodeId, {
    skip: !localNodeId,
  });
  const [updateLocalNode] = useUpdateLocalNodeMutation();
  const [addLocalNode] = useAddLocalNodeMutation();
  const [setDisableLocalNode] = useUpdateLocalNodeSetDisabledMutation();
  const [deleteLocalNode] = useUpdateLocalNodeSetDeletedMutation();

  const [localState, setLocalState] = useState({} as localNode);
  const [collapsedState, setCollapsedState] = useState(collapsed ?? false);
  const [showModalState, setShowModalState] = useState(false);
  const [deleteConfirmationTextInputState, setDeleteConfirmationTextInputState] = useState("");
  const [deleteEnabled, setDeleteEnabled] = useState(false);

  React.useEffect(() => {
    if (collapsed != undefined) {
      setCollapsedState(collapsed);
    }
  }, [collapsed]);

  const handleModalClose = () => {
    setShowModalState(false);
    setDeleteConfirmationTextInputState("");
    setDeleteEnabled(false);
  };

  const handleDeleteClick = () => {
    if (popoverRef.current) {
      (popoverRef.current as { close: Function }).close();
    }
    setShowModalState(true);
  };

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    submitNodeSettings();
  };

  const submitNodeSettings = () => {
    const form = new FormData();
    form.append("implementation", "LND");
    form.append("grpcAddress", localState.grpcAddress ?? "");
    if (localState.tlsFile) {
      form.append("tlsFile", localState.tlsFile, localState.tlsFileName);
    }
    if (localState.macaroonFile) {
      form.append("macaroonFile", localState.macaroonFile, localState.macaroonFileName);
    }
    // we are adding new node
    if (!localState.localNodeId) {
      addLocalNode(form);
      toastRef?.current?.addToast("Local node added", toastCategory.success);
      setLocalState({} as localNode);
      if (onAddSuccess) {
        onAddSuccess();
      }
      return;
    }
    updateLocalNode({ form, localNodeId: localState.localNodeId });
    toastRef?.current?.addToast("Local node info saved", toastCategory.success);
  };

  React.useEffect(() => {
    if (localNodeData) {
      setLocalState(localNodeData);
    }
  }, [localNodeData]);

  const handleTLSFileChange = (file: File) => {
    setLocalState({ ...localState, tlsFile: file, tlsFileName: file ? file.name : undefined });
  };

  const handleMacaroonFileChange = (file: File) => {
    setLocalState({ ...localState, macaroonFile: file, macaroonFileName: file ? file.name : undefined });
  };

  const handleAddressChange = (value: string) => {
    setLocalState({ ...localState, grpcAddress: value });
  };

  const handleCollapseClick = () => {
    setCollapsedState(!collapsedState);
  };

  const handleModalDeleteClick = () => {
    setShowModalState(false);
    setDeleteConfirmationTextInputState("");
    setDeleteEnabled(false);
    deleteLocalNode({ localNodeId: localState.localNodeId });
  };

  const handleDeleteConfirmationTextInputChange = (value: string) => {
    setDeleteConfirmationTextInputState(value);
    setDeleteEnabled(value.toLowerCase() === "delete");
  };

  const handleDisableClick = () => {
    setDisableLocalNode({ localNodeId: localState.localNodeId, disabled: !localState.disabled });
    if (popoverRef.current) {
      (popoverRef.current as { close: Function }).close();
    }
  };

  const implementationOptions = [{ value: "LND", label: "LND" } as SelectOption];

  const menuButton = <MoreIcon className={styles.moreIcon} />;
  return (
    <Box>
      <>
        {!addMode && (
          <div className={styles.header}>
            <div
              className={classNames(styles.connectionIcon, {
                [styles.connected]: true,
                [styles.disabled]: localState.disabled,
              })}
            >
              {!localState.disabled && <ConnectedIcon />}
              {localState.disabled && <DisconnectedIcon />}
            </div>
            <div className={styles.title}>{localNodeData?.grpcAddress}</div>
            <div className={classNames(styles.collapseIcon, { [styles.collapsed]: collapsedState })}>
              <ChevronIcon onClick={handleCollapseClick} />
            </div>
          </div>
        )}
        <Collapse collapsed={collapsedState} animate={!addMode}>
          <>
            {!addMode && (
              <>
                <div className={styles.borderSection}>
                  <div className={styles.detailHeader}>
                    <strong>Node Details</strong>
                    <Popover button={menuButton} className={"right"} ref={popoverRef}>
                      <div className={styles.nodeMenu}>
                        <Button
                          variant={buttonVariants.secondary}
                          text={localState.disabled ? "Enable node" : "Disable node"}
                          icon={localState.disabled ? <PlayIcon /> : <PauseIcon />}
                          onClick={handleDisableClick}
                        />
                        <Button
                          variant={buttonVariants.warning}
                          text={"Delete node"}
                          icon={<DeleteIcon />}
                          onClick={handleDeleteClick}
                        />
                      </div>
                    </Popover>
                  </div>
                </div>
              </>
            )}
            <div className={""}>
              <form onSubmit={handleSubmit}>
                <Select
                  label="Implementation"
                  onChange={() => {}}
                  options={implementationOptions}
                  value={implementationOptions.find((io) => io.value === localState?.implementation)}
                />
                <span id="address">
                  <TextInput
                    label="GRPC Address (IP or Tor)"
                    value={localState?.grpcAddress}
                    onChange={handleAddressChange}
                    placeholder="100.100.100.100:10009"
                  />
                </span>
                <span id="tls">
                  <File label="TLS Certificate" onFileChange={handleTLSFileChange} fileName={localState?.tlsFileName} />
                </span>
                <span id="macaroon">
                  <File
                    label="Macaroon"
                    onFileChange={handleMacaroonFileChange}
                    fileName={localState?.macaroonFileName}
                  />
                </span>
                <Button
                  variant={buttonVariants.secondary}
                  text={addMode ? "Add Node" : "Save node details"}
                  icon={<SaveIcon />}
                  onClick={submitNodeSettings}
                  fullWidth={true}
                />
              </form>
            </div>
          </>
        </Collapse>
        <Modal title={"Are you sure?"} icon={<DeleteIcon />} onClose={handleModalClose} show={showModalState}>
          <div className={styles.deleteConfirm}>
            <p>
              Deleting the node will prevent you from viewing it's data in Torq. Alternatively set node to disabled to
              simply stop the data subscription but keep data collected so far.
            </p>
            <p>
              This operation cannot be undone, type "<span className={styles.red}>delete</span>" to confirm.
            </p>

            <TextInput value={deleteConfirmationTextInputState} onChange={handleDeleteConfirmationTextInputChange} />
            <div className={styles.deleteConfirmButtons}>
              <a>Cancel</a>
              <Button
                variant={buttonVariants.warning}
                text={"Delete node"}
                icon={<DeleteIcon />}
                onClick={handleModalDeleteClick}
                disabled={!deleteEnabled}
              />
            </div>
          </div>
        </Modal>
      </>
    </Box>
  );
}
export default NodeSettings;
