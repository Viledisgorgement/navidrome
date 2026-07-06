import React from 'react'
import PropTypes from 'prop-types'
import { useTranslate } from 'react-admin'
import { Chip, makeStyles } from '@material-ui/core'

const useStyles = makeStyles((theme) => ({
  root: {
    display: 'flex',
    flexWrap: 'wrap',
    gap: theme.spacing(0.75),
    alignItems: 'center',
  },
}))

// Chip-based multi-select of users. An empty selection means "everyone".
const UserFilter = ({ users, selected, onChange }) => {
  const classes = useStyles()
  const translate = useTranslate()

  const toggleUser = (id) => {
    const next = selected.includes(id)
      ? selected.filter((u) => u !== id)
      : [...selected, id]
    onChange(next)
  }

  return (
    <div className={classes.root}>
      <Chip
        label={translate('community.allUsers')}
        size="small"
        color={selected.length === 0 ? 'primary' : 'default'}
        onClick={() => onChange([])}
      />
      {users.map((user) => (
        <Chip
          key={user.id}
          label={user.userName}
          size="small"
          color={selected.includes(user.id) ? 'primary' : 'default'}
          onClick={() => toggleUser(user.id)}
        />
      ))}
    </div>
  )
}

UserFilter.propTypes = {
  users: PropTypes.array.isRequired,
  selected: PropTypes.array.isRequired,
  onChange: PropTypes.func.isRequired,
}

export default UserFilter
