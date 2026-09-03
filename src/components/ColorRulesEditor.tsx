import React, { ChangeEvent } from 'react';
import { Button, IconButton, InlineField, InlineFieldRow, Input } from '@grafana/ui';
import { ColorRule } from '../types';

interface Props {
  rules: ColorRule[];
  onChange: (rules: ColorRule[]) => void;
}

export function ColorRulesEditor({ rules, onChange }: Props) {
  const updateRule = (index: number, patch: Partial<ColorRule>) => {
    onChange(rules.map((rule, i) => (i === index ? { ...rule, ...patch } : rule)));
  };

  const removeRule = (index: number) => {
    onChange(rules.filter((_, i) => i !== index));
  };

  const addRule = () => {
    onChange([...rules, { pattern: '', value: 0 }]);
  };

  return (
    <div>
      {rules.map((rule, index) => (
        <InlineFieldRow key={index}>
          <InlineField
            label="Stichwort"
            labelWidth={12}
            tooltip="Stichwort, nach dem im Termin-Titel gesucht wird (Groß-/Kleinschreibung egal)"
          >
            <Input
              width={30}
              placeholder="z.B. Urlaub"
              value={rule.pattern}
              onChange={(e: ChangeEvent<HTMLInputElement>) => updateRule(index, { pattern: e.currentTarget.value })}
            />
          </InlineField>
          <InlineField
            label="Wert"
            labelWidth={10}
            tooltip="Muss zu einer im Panel vorbereiteten Threshold-Stufe passen"
          >
            <Input
              type="number"
              width={12}
              value={rule.value}
              onChange={(e: ChangeEvent<HTMLInputElement>) =>
                updateRule(index, { value: e.currentTarget.value === '' ? 0 : Number(e.currentTarget.value) })
              }
            />
          </InlineField>
          <IconButton name="trash-alt" aria-label="Regel entfernen" tooltip="Regel entfernen" onClick={() => removeRule(index)} />
        </InlineFieldRow>
      ))}
      <Button icon="plus" variant="secondary" size="sm" onClick={addRule}>
        Regel hinzufügen
      </Button>
    </div>
  );
}
