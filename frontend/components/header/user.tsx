import styled from 'styled-components'
import {Button} from '@navikt/ds-react'
import {Success} from '@navikt/ds-icons'
import {UserData} from "./header";


const UserBox = styled.div``

interface userProps {
    user: UserData
}

export default function User({user}: userProps) {
    return (
        <UserBox>
            {user ?
                <div>
                    <Success style={{color: "#239f42", fontSize: '24px'}}/>Bobby Brown
                </div>
                :
                <Button key="logg-inn" variant="primary" size="small">Logg inn</Button>}
        </UserBox>
    )
}
