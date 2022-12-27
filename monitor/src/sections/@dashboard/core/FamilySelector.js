import * as React from 'react';
import PropTypes from 'prop-types';

import { FormControl, Autocomplete, TextField } from '@mui/material';

FamilySelector.propTypes = {
  type: PropTypes.string,
  items: PropTypes.arrayOf(PropTypes.string),
  defaultValue: PropTypes.string,
  handleChange: PropTypes.func,
};

export default function FamilySelector({ type, items, defaultValue, handleChange }) {
  const id = `${type}-select-family`;

  const [family, setFamily] = React.useState(defaultValue || '');

  const onChange = (event) => {
    setFamily(event.target.value);
    handleChange(event.target.value);
  };

  return (
    <FormControl fullWidth>
      <Autocomplete
        value={family}
        id={id}
        options={items}
        onChange={onChange}
        renderInput={(params) => <TextField {...params} label="Family" />}
      />
    </FormControl>
  );
}
