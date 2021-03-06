<?php

namespace App;

use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;

/**
 * @mixin Builder
 */
class Expense extends Model
{
    protected $fillable = ['amount', 'comment', 'category', 'date', 'user_id'];
    protected $casts    = ['amount' => 'float'];
    public $timestamps  = false;
    public $table       = "expenses";

    public function user()
    {
        return $this->belongsTo(User::class);
    }
}
