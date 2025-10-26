package com.afs.parser.model;

public class Transaction {
    private String date;
    private String postDate;
    private String description;
    private double amount;
    private String currency;
    private double amountInBHD;
    private boolean CR;

    public Transaction(){
        this.CR = false;
    }

    public String getDate() {
        return date;
    }

    public void setDate(String date) {
        this.date = date;
    }

    public String getDescription() {
        return description;
    }

    public void setDescription(String description) {
        this.description = description;
    }

    public double getAmount() {
        return amount;
    }

    public void setAmount(Double amount) {
        this.amount = amount;
    }

    public double getAmountInBHD() {
        return amountInBHD;
    }

    public void setAmountInBHD(double amountInBHD) {
        this.amountInBHD = amountInBHD;
    }

    public String getCurrency() {
        return currency;
    }

    public void setCurrency(String currency) {
        this.currency = currency;
    }

    public boolean isCR() {
        return CR;
    }

    public void setCR(boolean CR) {
        this.CR = CR;
    }

    public String getPostDate() {
        return postDate;
    }

    public void setPostDate(String postDate) {
        this.postDate = postDate;
    }
}
