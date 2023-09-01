import React, {useEffect, useState} from 'react';

import Button from 'react-bootstrap/Button';
import Modal from 'react-bootstrap/Modal';
import Table from 'react-bootstrap/Table';
import Form from 'react-bootstrap/Form';
import {OverlayTrigger, Tooltip} from 'react-bootstrap';

import './styles.scss';

type Props = {
    visibility: boolean;
    setVisibility: React.Dispatch<React.SetStateAction<boolean>>;
    config: Configs[];
    configIndex: number;
}

const ActionModal = ({visibility, setVisibility, config, configIndex}: Props) => {
    const [attachmentMessageAvailable, setAttachmentMessageAvailable] = useState(false);

    const actionsLength = config[configIndex]?.actions?.length;

    const checkAttachmentMessage = () => {
        if (config[configIndex]?.attachmentMessage?.length) {
            setAttachmentMessageAvailable(Boolean(config[configIndex]?.attachmentMessage?.[0]));
            return;
        }

        setAttachmentMessageAvailable(false);
    };

    useEffect(() => {
        checkAttachmentMessage();
    }, []);

    const handleClose = () => setVisibility(false);

    return (
        <Modal
            className='custom-modal'
            show={visibility}
            onHide={handleClose}
        >
            <Modal.Header closeButton={false}>
                <Modal.Title>{'Actions'}</Modal.Title>
            </Modal.Header>

            <Modal.Body className='custom-modal-body'>
                {attachmentMessageAvailable || (config[configIndex]?.actions && actionsLength) ? (<>
                    {attachmentMessageAvailable ? (
                        <Form>
                            <Form.Group className='form-group'>
                                <Form.Label>{'Attachment Message'}</Form.Label>
                                <Form.Control
                                    type='long-text'
                                    value={config[configIndex].attachmentMessage?.join(',') ?? ''}
                                    placeholder=''
                                    aria-label='Disabled input example'
                                    readOnly={true}
                                />
                            </Form.Group>
                        </Form>
                    ) : (<p>{'No attachment message configured'}</p>)
                    }
                    {config[configIndex]?.actions && actionsLength ? (
                        <div>
                            <Form>
                                <Form.Group className='action-group'>
                                    <Form.Label>{'Actions'}</Form.Label>
                                </Form.Group>
                            </Form>
                            <div className='list-table'>
                                <Table
                                    striped={true}
                                >
                                    <thead>
                                        <tr>
                                            <th className='type-action'>{'Type'}</th>
                                            <th className='display-name-action'>{'Display Name'}</th>
                                            <th className='name-action'>{'Name'}</th>
                                            <th className='channels-added-action'>{'Add to Channels'}</th>
                                            <th className='success-message-action'>{'Success Message'}</th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        {
                                    config[configIndex].actions?.map((val, i) =>
                                        (
                                            <tr key={i.toString()}>
                                                <td className='type-action'>
                                                    <OverlayTrigger
                                                        placement='top'
                                                        overlay={<Tooltip>{val.actionType}</Tooltip>}
                                                    >
                                                        <p>
                                                            {val.actionType}
                                                        </p>
                                                    </OverlayTrigger>
                                                </td>
                                                <td className='display-name-action'>
                                                    <OverlayTrigger
                                                        placement='top'
                                                        overlay={<Tooltip>{val.actionDisplayName}</Tooltip>}
                                                    >
                                                        <p className='display-name-content'>
                                                            {val.actionDisplayName}
                                                        </p>
                                                    </OverlayTrigger>
                                                </td>
                                                <td className='name-action'>
                                                    <OverlayTrigger
                                                        placement='top'
                                                        overlay={<Tooltip>{val.actionName}</Tooltip>}
                                                    >
                                                        <p className='action-name-content'>
                                                            {val.actionName}
                                                        </p>
                                                    </OverlayTrigger>
                                                </td>
                                                <td className='channels-added-action'>
                                                    <OverlayTrigger
                                                        placement='top'
                                                        overlay={<Tooltip>{val.channelsAddedTo.join(', ')}</Tooltip>}
                                                    >
                                                        <p className='channels-added-content'>
                                                            {val.channelsAddedTo.join(', ')}
                                                        </p>
                                                    </OverlayTrigger>
                                                </td>
                                                <td className='success-message-action'>
                                                    <OverlayTrigger
                                                        placement='top'
                                                        overlay={<Tooltip>{val.actionSuccessfullMessage.join(',')}</Tooltip>}
                                                    >
                                                        <p className='successfull-message-content'>
                                                            {val.actionSuccessfullMessage.join(',')}
                                                        </p>
                                                    </OverlayTrigger>
                                                </td>
                                            </tr>
                                        ),
                                    )
                                        }
                                    </tbody>
                                </Table>
                            </div>
                        </div>
                    ) : (<p>{'No action configured'}</p>)
                    }
                </>) : (<p>{'No attachment message or action configured'}</p>)}
            </Modal.Body>

            <Modal.Footer>
                <Button
                    variant='secondary'
                    onClick={handleClose}
                >{'Close'}</Button>
            </Modal.Footer>
        </Modal>
    );
};

export default ActionModal;
